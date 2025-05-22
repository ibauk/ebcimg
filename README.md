# ebcimg
An image handler for use with ebcfetch (ScoreMaster)

Usage: ebcimg *infile* *outfile*

This reads the input file and converts it to JPG format. It can handle JPG, PNG and HEIC formats. 

If the image can't be decoded, an image is created
explaining that the file can't be decoded. 

Outputs, regardless of filename, are always in JPG format.

