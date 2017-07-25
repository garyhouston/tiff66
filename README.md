# tiff66
tiff66 is a Go library for encoding and decoding TIFF files. It can be used to extract or add information to TIFF files, but doesn't include functionality for processing images.

For documentation, see https://godoc.org/github.com/garyhouston/tiff66.

## Notes and limitations
Data is encoded and decoded from Go byte slices, so is limited to files that can fit in available memory. TIFF files can be up to 4GB in size. Reading and rewriting a file will require space for two byte slices.

Data is unpacked into structures that contain pointers to the raw data in the original byte slices. This saves copying and memory use, but modifying the data in one place will also modify it in the other. The buffer could be modified in-place if only simple changes to field data are made.

The tiff66print program prints the IFDs (image file directories) and fields of a TIFF file.

The tiff66repack program decodes a TIFF file and encodes it into a new file.

The [Exif44](https://github.com/garyhouston/exif44) library extends this library with additional support for Exif fields, and has corresponding print and repack programs.

This library is still under construction and may change at any moment without backwards compatibility.

TIFF is a difficult file format, and there may be omissions in this library that prevent correct processing of all possible TIFF files. For example, fields that are apparently integers can actually be pointers to arbitrary data. Such fields need to be supported in the library explicitly. The output of tiff66print will show any unknown fields. The sizes of the original and repacked files can also be compared. The repacked version may be larger if mmore than one TIFF field points to the same data; encoding will duplicate it. Output from tiff66print can also be compared between the original file and the repacked version. Some differences are to be expected, such as positions of sub-IFDs. 

Exif blocks are in TIFF format, but may contain proprietary maker notes. Currently, only Nikon and Panasonic maker notes are encoded and decoded. In some cases, unsupported maker notes will be broken if the Exif block is repacked, since they contain pointers that would need adjustment.

This library makes no provision for modification of data in multiple threads. Mutexes etc., should be used as required.

'66' is an arbitrary number to distinguish this library from all the other TIFF libraries.
