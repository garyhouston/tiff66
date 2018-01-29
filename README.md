# tiff66
tiff66 is a Golang library for encoding and decoding TIFF files. It can be used to extract or add information to TIFF files, but doesn't include functionality for processing images.

For documentation, see https://godoc.org/github.com/garyhouston/tiff66.

## Notes and limitations
This library is still under construction and may change at any moment without backwards compatibility.

Data is encoded and decoded from Go byte slices, so is limited to files that can fit in available memory. TIFF files can be up to 4GB in size. Reading and rewriting a file will require space for two byte slices.

Data is unpacked into structures that contain pointers to the raw data in the original byte slices. This saves copying and memory use, but modifying the data in one place will also modify it in the other. The buffer could be modified in-place if only simple changes to field data are made.

The tiff66print program prints the IFDs (image file directories) and fields of a TIFF file.

The tiff66repack program decodes a TIFF file and encodes it into a new file.

The [Exif44](https://github.com/garyhouston/exif44) library extends this library with additional support for Exif fields, and has corresponding print and repack programs.

TIFF is a difficult file format, and there may be omissions in this library that prevent correct processing of all possible TIFF files. For example, fields that are apparently integers can actually be pointers to arbitrary data. Such fields need to be supported in the library explicitly if the data is to be retained when rewritten. The output of tiff66print will show any unknown fields. The sizes of the original and repacked files can also be compared. The repacked version may be larger if more than one TIFF field points to the same data; encoding will duplicate it. Output from tiff66print can also be compared between the original file and the repacked version. Some differences are to be expected, such as positions of sub-IFDs. 

Exif blocks are in TIFF format, but may contain proprietary maker notes. Currently, Canon, Fujifilm, Nikon, Olympus and Panasonic maker notes can be encoded and decoded. Some Sony maker notes are partly decoded, but may be broken if rewritten. In some cases, unsupported maker notes will be broken if the Exif block is rewritten, since they contain pointers that would need adjustment.

Canon maker notes (possibly from the EOS 300D only) may contain a PreviewImageInfo field, which refers to the position and length of a preview image. Since the image is located outside the JPEG block that contains the maker note, special processing would be needed to preserve it when rewriting a file.

No provision is made for modification of data in multiple threads. Mutexes etc., should be used as required.

When reading files, GetIFDTree will attempt to decode as much data as possible, even if errors occur. If multiple errors are encountered, they will be encoded in a [multierror](https://github.com/hashicorp/go-multierror) structure.

Information about maker note formats was obtained from [Exiftool](https://www.sno.phy.queensu.ca/~phil/exiftool/).

'66' is an arbitrary number to distinguish this library from all the other TIFF libraries.
