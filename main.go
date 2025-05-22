package main

/*
 *
 * ebcimg - image processor for Electronic Bonus Claiming
 *
 * Shamelessly stolen from github/adrium/goheif for the purpose initially of
 * converting HEIC images to JPGs.
 *
 * Built into a specialist handler for use with ScoreMaster
 *
 */
import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	_ "embed"

	"github.com/adrium/goheif"
)

const programversion = "ebcimg v1.1 - Image helper for ScoreMaster"

var showVersion = flag.Bool("v", false, "Show version info")

//go:embed font.ttf
var fontBytes []byte

var Options = jpeg.Options{Quality: 100}

// Skip Writer for exif writing
type writerSkipper struct {
	w           io.Writer
	bytesToSkip int
}

func (w *writerSkipper) Write(data []byte) (int, error) {
	if w.bytesToSkip <= 0 {
		return w.w.Write(data)
	}

	if dataLen := len(data); dataLen < w.bytesToSkip {
		w.bytesToSkip -= dataLen
		return dataLen, nil
	}

	if n, err := w.w.Write(data[w.bytesToSkip:]); err == nil {
		n += w.bytesToSkip
		w.bytesToSkip = 0
		return n, nil
	} else {
		return n, err
	}
}

func newWriterExif(w io.Writer, exif []byte) (io.Writer, error) {
	writer := &writerSkipper{w, 2}
	soi := []byte{0xff, 0xd8}
	if _, err := w.Write(soi); err != nil {
		return nil, err
	}

	if exif != nil {
		app1Marker := 0xe1
		markerlen := 2 + len(exif)
		marker := []byte{0xff, uint8(app1Marker), uint8(markerlen >> 8), uint8(markerlen & 0xff)}
		if _, err := w.Write(marker); err != nil {
			return nil, err
		}

		if _, err := w.Write(exif); err != nil {
			return nil, err
		}
	}

	return writer, nil
}

func isHeic(fi *os.File, fo *os.File) bool {

	fi.Seek(0, 0)
	exif, _ := goheif.ExtractExif(fi)

	img, err := goheif.Decode(fi)
	if err != nil {
		return false
	}

	w, _ := newWriterExif(fo, exif)
	err = jpeg.Encode(w, img, &Options)
	return err == nil

}

func isJpg(fi *os.File, fo *os.File) bool {

	fi.Seek(0, 0)
	img, err := jpeg.Decode(fi)
	if err != nil {
		return false
	}
	err = jpeg.Encode(fo, img, &Options)
	return err == nil

}
func isPng(fi *os.File, fo *os.File) bool {

	fi.Seek(0, 0)
	img, err := png.Decode(fi)
	if err != nil {
		return false
	}
	err = jpeg.Encode(fo, img, &Options)
	return err == nil

}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stderr, programversion)
	}
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: ebcimg <in-file> <out-file>")
		os.Exit(0) // Not necessarily an error, might be Ebcfetch testing for runability
	}

	fin, fout := flag.Arg(0), flag.Arg(1)

	fi, err := os.Open(fin)
	if err != nil {
		log.Fatal(err)
	}
	defer fi.Close()

	fo, err := os.OpenFile(fout, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Failed to create output file %s: %v\n", fout, err)
		return
	}
	defer fo.Close()

	if isJpg(fi, fo) {
		log.Printf("%v is a JPG\n", fin)
		return
	}
	if isPng(fi, fo) {
		log.Printf("%v is a PNG\n", fin)
		return
	}
	if isHeic(fi, fo) {
		log.Printf("%v is a HEIC\n", fin)
		return
	}

	makeFailImage(fo, filepath.Base(fin))

}

func makeFailImage(f *os.File, x string) {

	log.Printf("Making fail image for %v\n", x)
	msg := "Image file cannot be decoded!"
	msg2 := "Please refer to the original email."
	width := 360
	height := 240

	upLeft := image.Point{0, 0}
	lowRight := image.Point{width, height}

	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})

	// Colors are defined by Red, Green, Blue, Alpha uint8 values.
	cyan := color.RGBA{100, 200, 200, 0xff}

	// Set color for each pixel.
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, cyan)
		}
	}

	if err := addText(img, msg, image.Point{15, 50}, color.Black, 24); err != nil {
		log.Fatalf("Error adding text: %v", err)
	}
	addText(img, x, image.Point{15, 100}, color.Black, 18)
	addText(img, msg2, image.Point{15, 150}, color.Black, 18)
	addText(img, programversion, image.Point{15, 200}, color.Black, 12)

	err := jpeg.Encode(f, img, &Options)
	if err != nil {
		log.Fatalf("Can't encode %v", err)
	}
}

func addText(baseImage *image.RGBA, text string, point image.Point, col color.Color, fontSize float64) error {

	ttf, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Fatalf("Can't parse font %v", err)
		return err
	}

	face, err := opentype.NewFace(ttf, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}

	drawer := &font.Drawer{
		Dst:  baseImage,
		Src:  image.NewUniform(col),
		Face: face,
		Dot: fixed.Point26_6{
			X: fixed.I(point.X),
			Y: fixed.I(point.Y),
		},
	}

	drawer.DrawString(text)

	return nil
}
