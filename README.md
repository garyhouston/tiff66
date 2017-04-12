# tiff66
tiff66 is a Go library for encoding and decoding TIFF files. It can be used to extract or add information to TIFF files, but doesn't include functionality for processing images.

For documentation, see https://godoc.org/github.com/garyhouston/tiff66.

## Notes and limitations
Data is encoded and decoded from Go byte slices, so is limited to files that can fit in available memory. TIFF files can be up to 4GB in size. Reading and rewriting a file will require space for two byte slices.

Data is unpacked into structures that contain pointers to the raw data in the original byte slices. This saves copying and memory use, but modifying the data in one place will also modify it in the other. The buffer could be modified in-place if only simple changes to field data are made.

The tiff66print program prints the IFDs (image file directories) and fields of a TIFF file.

The tiff66repack program decodes a TIFF file and encodes it into a new file.

TIFF is a difficult file format, and there may be omissions in this library that prevent correct processing of all possible TIFF files. For example, if an unsupported data field holds a pointer (LONG value) to data, that data will be omitted. The output of tiff66print will show any unknown fields. tiff66repack can be used to check if the decoding and encoding process changes a file's size significantly. Files can also grow in size: if more than one TIFF field refers to the same data, encoding will duplicate it.

In addition, Makernote fields found in Exif are not currently decoded, and are likely to contain pointers. The pointers will be broken if the file is repacked.

This library makes no provision for modification of data in multiple threads. Mutexes etc., should be used as required.

'66' is an arbitrary number to distinguish this library from all the other TIFF libraries.
