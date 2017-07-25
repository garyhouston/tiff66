package tiff66

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

type Type uint8

// TIFF data types (uppercase as in the TIFF spec).
const (
	BYTE      Type = 1
	ASCII     Type = 2
	SHORT     Type = 3
	LONG      Type = 4
	RATIONAL  Type = 5
	SBYTE     Type = 6
	UNDEFINED Type = 7
	SSHORT    Type = 8
	SLONG     Type = 9
	SRATIONAL Type = 10
	FLOAT     Type = 11
	DOUBLE    Type = 12
	IFD       Type = 13 // Supplement 1
)

var TypeNames = map[Type]string{
	BYTE:      "Byte",
	ASCII:     "ASCII",
	SHORT:     "Short",
	LONG:      "Long",
	RATIONAL:  "Rational",
	SBYTE:     "SByte",
	UNDEFINED: "Undefined",
	SSHORT:    "SShort",
	SLONG:     "SLong",
	SRATIONAL: "SRational",
	FLOAT:     "Float",
	DOUBLE:    "Double",
	IFD:       "IFD",
}

// Return the name of a TIFF type.
func (t Type) Name() string {
	name, found := TypeNames[t]
	if found {
		return name
	} else {
		return "Unknown"
	}
}

// Byte size of a single value of each TIFF type.
var TypeSizes = map[Type]uint32{
	BYTE:      1,
	ASCII:     1,
	SHORT:     2,
	LONG:      4,
	RATIONAL:  8,
	SBYTE:     1,
	UNDEFINED: 1,
	SSHORT:    2,
	SLONG:     4,
	SRATIONAL: 8,
	FLOAT:     4,
	DOUBLE:    8,
	IFD:       4,
}

// Return the size of a single value of a TIFF type.
func (t Type) Size() uint32 {
	size, found := TypeSizes[t]
	if found {
		return size
	} else {
		return 0
	}
}

// Indicate if the given type is one of the TIFF integer types.
func (t Type) IsIntegral() bool {
	return t == BYTE || t == SHORT || t == LONG || t == SBYTE || t == SSHORT || t == SLONG
}

// Indicate if the given type is one of the TIFF rational types.
func (t Type) IsRational() bool {
	return t == RATIONAL || t == SRATIONAL
}

// Indicate if the given type is one of the TIFF floating point types.
func (t Type) IsFloat() bool {
	return t == FLOAT || t == DOUBLE
}

type Tag uint16

// Some of the tags that may be found in TIFF main IFDs (not alternative
// or private IFDs such as Exif.) Tags are from TIFF 6.0 if not otherwise
// specified.
const (
	NewSubfileType              = 0x0FE
	SubfileType                 = 0x0FF
	ImageWidth                  = 0x100
	ImageLength                 = 0x101
	BitsPerSample               = 0x102
	Compression                 = 0x103
	PhotometricInterpretation   = 0x106
	Threshholding               = 0x107
	CellWidth                   = 0x108
	CellLength                  = 0x109
	FillOrder                   = 0x10A
	DocumentName                = 0x10D
	ImageDescription            = 0x10E
	Make                        = 0x10F
	Model                       = 0x110
	StripOffsets                = 0x111
	Orientation                 = 0x112
	SamplesPerPixel             = 0x115
	RowsPerStrip                = 0x116
	StripByteCounts             = 0x117
	MinSampleValue              = 0x118
	MaxSampleValue              = 0x119
	XResolution                 = 0x11A
	YResolution                 = 0x11B
	PlanarConfiguration         = 0x11C
	PageName                    = 0x11D
	XPosition                   = 0x11E
	YPosition                   = 0x11F
	FreeOffsets                 = 0x120
	FreeByteCounts              = 0x121
	GrayResponseUnit            = 0x122
	GrayResponseCurve           = 0x123
	T4Options                   = 0x124
	T6Options                   = 0x125
	ResolutionUnit              = 0x128
	PageNumber                  = 0x129
	TransferFunction            = 0x12D
	Software                    = 0x131
	DateTime                    = 0x132
	Artist                      = 0x13B
	HostComputer                = 0x13C
	Predictor                   = 0x13D
	WhitePoint                  = 0x13E
	PrimaryChromaticities       = 0x13F
	ColorMap                    = 0x140
	HalftoneHints               = 0x141
	TileWidth                   = 0x142
	TileLength                  = 0x143
	TileOffsets                 = 0x144
	TileByteCounts              = 0x145
	BadFaxLines                 = 0x146 // TIFF F (RFC 2306)
	CleanFaxData                = 0x147 // TIFF F (RFC 2306)
	ConsecutiveBadFaxLines      = 0x148 // TIFF F (RFC 2306)
	SubIFDs                     = 0x14A // Supplement 1
	InkSet                      = 0x14C
	InkNames                    = 0x14D
	NumberOfInks                = 0x14E
	DotRange                    = 0x150
	TargetPrinter               = 0x151
	ExtraSamples                = 0x152
	SampleFormat                = 0x153
	SMinSampleValue             = 0x154
	SMaxSampleValue             = 0x155
	TransferRange               = 0x156
	ClipPath                    = 0x157 // Supplement 1
	XClipPathUnits              = 0x158 // Supplement 1
	YClipPathUnits              = 0x159 // Supplement 1
	Indexed                     = 0x15A // Supplement 1
	JPEGTables                  = 0x15B // Supplement 2
	OPIProxy                    = 0x15F // Supplement 1
	JPEGProc                    = 0x200
	JPEGInterchangeFormat       = 0x201
	JPEGInterchangeFormatLength = 0x202
	JPEGRestartInterval         = 0x203
	JPEGLosslessPredictors      = 0x205
	JPEGPointTransforms         = 0x206
	JPEGQTables                 = 0x207
	JPEGDCTables                = 0x208
	JPEGACTables                = 0x209
	YCbCrCoefficients           = 0x211
	YCbCrSubSampling            = 0x212
	YCbCrPositioning            = 0x213
	ReferenceBlackWhite         = 0x214
	XMP                         = 0x2BC  // XMP part 3
	ImageID                     = 0x800  // Supplement 1
	PrintIM                     = 0xC4A5 // Epson print image matching
	Copyright                   = 0x8298
	ModelPixelScaleTag          = 0x830E // GeoTIFF
	IPTC                        = 0x83BB // Mentioned in XMP part 3
	ModelTiepointTag            = 0x8482 // GeoTIFF
	ModelTransformationTag      = 0x85D8 // GeoTIFF
	PSIR                        = 0x8649 // Photoshop Image Resources, Mentioned in XMP part 3
	ExifIFD                     = 0x8769 // Exif 2.3
	ICCProfile                  = 0x8773 // ICC.1:2003-09
	GeoKeyDirectoryTag          = 0x87AF // GeoTIFF
	GeoDoubleParamsTag          = 0x87B0 // GeoTIFF
	GeoAsciiParamsTag           = 0x87B1 // GeoTIFF
	GPSIFD                      = 0x8825 // Exif 2.3
	ImageSourceData             = 0x935C // Supplement 2
)

// Mappings from TIFF tags to strings.
var TagNames = map[Tag]string{
	NewSubfileType:              "NewSubfileType",
	SubfileType:                 "SubfileType",
	ImageWidth:                  "ImageWidth",
	ImageLength:                 "ImageLength",
	BitsPerSample:               "BitsPerSample",
	Compression:                 "Compression",
	PhotometricInterpretation:   "PhotometricInterpretation",
	Threshholding:               "Threshholding",
	CellWidth:                   "CellWidth",
	CellLength:                  "CellLength",
	FillOrder:                   "FillOrder",
	DocumentName:                "DocumentName",
	ImageDescription:            "ImageDescription",
	Make:                        "Make",
	Model:                       "Model",
	StripOffsets:                "StripOffsets",
	Orientation:                 "Orientation",
	SamplesPerPixel:             "SamplesPerPixel",
	RowsPerStrip:                "RowsPerStrip",
	StripByteCounts:             "StripByteCounts",
	MinSampleValue:              "MinSampleValue",
	MaxSampleValue:              "MaxSampleValue",
	XResolution:                 "XResolution",
	YResolution:                 "YResolution",
	PlanarConfiguration:         "PlanarConfiguration",
	PageName:                    "PageName",
	XPosition:                   "XPosition",
	YPosition:                   "YPosition",
	FreeOffsets:                 "FreeOffsets",
	FreeByteCounts:              "FreeByteCounts",
	GrayResponseUnit:            "GrayResponseUnit",
	GrayResponseCurve:           "GrayResponseCurve",
	T4Options:                   "T4Options",
	T6Options:                   "T6Options",
	ResolutionUnit:              "ResolutionUnit",
	PageNumber:                  "PageNumber",
	TransferFunction:            "TransferFunction",
	Software:                    "Software",
	DateTime:                    "DateTime",
	Artist:                      "Artist",
	HostComputer:                "HostComputer",
	Predictor:                   "Predictor",
	WhitePoint:                  "WhitePoint",
	PrimaryChromaticities:       "PrimaryChromaticities",
	ColorMap:                    "ColorMap",
	HalftoneHints:               "HalftoneHints",
	TileWidth:                   "TileWidth",
	TileLength:                  "TileLength",
	TileOffsets:                 "TileOffsets",
	TileByteCounts:              "TileByteCounts",
	BadFaxLines:                 "BadFaxLines",
	CleanFaxData:                "CleanFaxData",
	ConsecutiveBadFaxLines:      "ConsecutiveBadFaxLines",
	SubIFDs:                     "SubIFDs",
	InkSet:                      "InkSet",
	InkNames:                    "InkNames",
	NumberOfInks:                "NumberOfInks",
	DotRange:                    "DotRange",
	TargetPrinter:               "TargetPrinter",
	ExtraSamples:                "ExtraSamples",
	SampleFormat:                "SampleFormat",
	SMinSampleValue:             "SMinSampleValue",
	SMaxSampleValue:             "SMaxSampleValue",
	TransferRange:               "TransferRange",
	ClipPath:                    "ClipPath",
	XClipPathUnits:              "XClipPathUnits",
	YClipPathUnits:              "YClipPathUnits",
	Indexed:                     "Indexed",
	JPEGTables:                  "JPEGTables",
	OPIProxy:                    "OPIProxy",
	JPEGProc:                    "JPEGProc",
	JPEGInterchangeFormat:       "JPEGInterchangeFormat",
	JPEGInterchangeFormatLength: "JPEGInterchangeFormatLength",
	JPEGRestartInterval:         "JPEGRestartInterval",
	JPEGLosslessPredictors:      "JPEGLosslessPredictors",
	JPEGPointTransforms:         "JPEGPointTransforms",
	JPEGQTables:                 "JPEGQTables",
	JPEGDCTables:                "JPEGDCTables",
	JPEGACTables:                "JPEGACTables",
	YCbCrCoefficients:           "YCbCrCoefficients",
	YCbCrSubSampling:            "YCbCrSubSampling",
	YCbCrPositioning:            "YCbCrPositioning",
	ReferenceBlackWhite:         "ReferenceBlackWhite",
	XMP:                         "XMP",
	ImageID:                     "ImageID",
	PrintIM:                     "PrintIM",
	Copyright:                   "Copyright",
	ModelPixelScaleTag:          "ModelPixelScaleTag",
	IPTC:                        "IPTC",
	ModelTiepointTag:            "ModelTiepointTag",
	ModelTransformationTag:      "ModelTransformationTag",
	PSIR:               "PSIR",
	ExifIFD:            "ExifIFD",
	ICCProfile:         "ICCProfile",
	GeoKeyDirectoryTag: "GeoKeyDirectoryTag",
	GeoDoubleParamsTag: "GeoDoubleParamsTag",
	GeoAsciiParamsTag:  "GeoAsciiParamsTag",
	GPSIFD:             "GPSIFD",
	ImageSourceData:    "ImageSourceData",
}

// A TIFF field; an IFD entry and its data.
type Field struct {
	Tag   Tag
	Type  Type
	Count uint32
	Data  []byte
}

// Field data size.
func (f Field) Size() uint32 {
	return f.Type.Size() * f.Count
}

// Indicate if this field a pointer to another IFD. Depends on the IFD
// namespace.
func (f Field) IsIFD(space TagSpace) bool {
	if f.Type == IFD {
		return true
	}
	switch space {
	case TIFFSpace:
		return f.Tag == SubIFDs || f.Tag == ExifIFD || f.Tag == GPSIFD
	case ExifSpace:
		return f.Tag == interOpIFD
	case Nikon2Space:
		return f.Tag == Nikon2PreviewIFD || f.Tag == Nikon2NikonScanIFD
	default:
		return false
	}
}

// Return a BYTE field's ith data element.
func (f Field) Byte(i uint32) uint8 {
	return f.Data[i]
}

// Set a BYTE field's ith data element.
func (f Field) PutByte(val uint8, i uint32) {
	f.Data[i] = val
}

// Return a SHORT field's ith data element.
func (f Field) Short(i uint32, order binary.ByteOrder) uint16 {
	return order.Uint16(f.Data[i*2:])
}

// Set a SHORT field's ith data element.
func (f Field) PutShort(val uint16, i uint32, order binary.ByteOrder) {
	order.PutUint16(f.Data[i*2:], val)
}

// Return a LONG field's ith data element.
func (f Field) Long(i uint32, order binary.ByteOrder) uint32 {
	return order.Uint32(f.Data[i*4:])
}

// Set a LONG field's ith data element.
func (f Field) PutLong(val uint32, i uint32, order binary.ByteOrder) {
	order.PutUint32(f.Data[i*4:], val)
}

// Return a SBYTE field's ith data element.
func (f Field) SByte(i uint32) int8 {
	return int8(f.Data[i])
}

// Set a SBYTE field's ith data element.
func (f Field) PutSByte(val int8, i uint32) {
	f.Data[i] = uint8(val)
}

// Return a SSHORT field's ith data element.
func (f Field) SShort(i uint32, order binary.ByteOrder) int16 {
	return int16(order.Uint16(f.Data[i*2:]))
}

// Set a SSHORT field's ith data element.
func (f Field) PutSShort(val int16, i uint32, order binary.ByteOrder) {
	order.PutUint16(f.Data[i*2:], uint16(val))
}

// Return a LONG field's ith data element.
func (f Field) SLong(i uint32, order binary.ByteOrder) int32 {
	return int32(order.Uint32(f.Data[i*4:]))
}

// Set a LONG field's ith data element.
func (f Field) PutSLong(val int32, i uint32, order binary.ByteOrder) {
	order.PutUint32(f.Data[i*4:], uint32(val))
}

// Return an integral-valued field's ith data element.
func (f Field) AnyInteger(i uint32, order binary.ByteOrder) int64 {
	switch f.Type {
	case BYTE:
		return int64(f.Byte(i))
	case SHORT:
		return int64(f.Short(i, order))
	case LONG:
		return int64(f.Long(i, order))
	case SBYTE:
		return int64(f.SByte(i))
	case SSHORT:
		return int64(f.SShort(i, order))
	case SLONG:
		return int64(f.SLong(i, order))
	}
	panic("AnyInteger called with wrong type field")
}

// Set an integral-valued field's ith data element.
func (f Field) PutAnyInteger(val int64, i uint32, order binary.ByteOrder) {
	switch f.Type {
	case BYTE:
		f.PutByte(uint8(val), i)
	case SHORT:
		f.PutShort(uint16(val), i, order)
	case LONG:
		f.PutLong(uint32(val), i, order)
	case SBYTE:
		f.PutSByte(int8(val), i)
	case SSHORT:
		f.PutSShort(int16(val), i, order)
	case SLONG:
		f.PutSLong(int32(val), i, order)
	default:
		panic("PutAnyInteger called with wrong type field")
	}
}

// Return a RATIONAL field's ith data element.
func (f Field) Rational(i uint32, order binary.ByteOrder) (uint32, uint32) {
	return order.Uint32(f.Data[i*8:]), order.Uint32(f.Data[i*8+4:])
}

// Set a RATIONAL field's ith data element.
func (f Field) PutRational(n uint32, d uint32, i uint32, order binary.ByteOrder) {
	order.PutUint32(f.Data[i*8:], n)
	order.PutUint32(f.Data[i*8+4:], d)
}

// Return a SRATIONAL field's ith data element.
func (f Field) SRational(i uint32, order binary.ByteOrder) (int32, int32) {
	return int32(order.Uint32(f.Data[i*8:])), int32(order.Uint32(f.Data[i*8+4:]))
}

// Set a SRATIONAL field's ith data element.
func (f Field) PutSRational(n int32, d int32, i uint32, order binary.ByteOrder) {
	order.PutUint32(f.Data[i*8:], uint32(n))
	order.PutUint32(f.Data[i*8+4:], uint32(d))
}

// Return a rational-valued field's ith data element.
func (field Field) AnyRational(i uint32, order binary.ByteOrder) (int64, int64) {
	switch field.Type {
	case RATIONAL:
		n, d := field.Rational(i, order)
		return int64(n), int64(d)
	case SRATIONAL:
		n, d := field.SRational(i, order)
		return int64(n), int64(d)
	}
	panic("AnyRational called with wrong type field")
}

// Set a rational-valued field's ith data element.
func (field Field) PutAnyRational(n int64, d int64, i uint32, order binary.ByteOrder) {
	switch field.Type {
	case RATIONAL:
		field.PutRational(uint32(n), uint32(d), i, order)
	case SRATIONAL:
		field.PutSRational(int32(n), int32(d), i, order)
	}
	panic("PutAnyRational called with wrong type field")
}

// Return a FLOAT field's ith data element.
func (f Field) Float(i uint32, order binary.ByteOrder) float32 {
	bits := order.Uint32(f.Data[i*4:])
	return math.Float32frombits(bits)
}

// Set a FLOAT field's ith data element.
func (f Field) PutFloat(val float32, i uint32, order binary.ByteOrder) {
	order.PutUint32(f.Data[i*4:], math.Float32bits(val))
}

// Return a DOUBLE field's ith data element.
func (f Field) Double(i uint32, order binary.ByteOrder) float64 {
	bits := order.Uint64(f.Data[i*8:])
	return math.Float64frombits(bits)
}

// Set a DOUBLE field's ith data element.
func (f Field) PutDouble(val float64, i uint32, order binary.ByteOrder) {
	order.PutUint64(f.Data[i*8:], math.Float64bits(val))
}

// Return a floating point field's ith data element.
func (f Field) AnyFloat(i uint32, order binary.ByteOrder) float64 {
	switch f.Type {
	case FLOAT:
		return float64(f.Float(i, order))
	case DOUBLE:
		return f.Double(i, order)
	}
	panic("AnyFloat called with wrong type field")
}

// Set a floating point field's ith data element.
func (f Field) PutAnyFloat(val float64, i uint32, order binary.ByteOrder) {
	switch f.Type {
	case FLOAT:
		f.PutFloat(float32(val), i, order)
	case DOUBLE:
		f.PutDouble(val, i, order)
	}
	panic("PutAnyFloat called with wrong type field")
}

// Return an ASCII field data as a string. It omits the terminating NUL if
// present but retains any other NULs.
func (f Field) ASCII() string {
	if f.Data[len(f.Data)-1] == 0 {
		return string(f.Data[:len(f.Data)-1])
	} else {
		return string(f.Data)
	}
}

// Set an ASCII field data from a string, including a trailing NUL. The
// field's data will be reallocated.
func (f *Field) PutASCII(val string) {
	f.Data = make([]byte, len(val)+1)
	copy(f.Data, val)
	f.Data[len(val)] = 0
}

// Helper for Field.Print: print a field's data values.
func printValues(f Field, order binary.ByteOrder, limit uint32, print func(Field, uint32, binary.ByteOrder)) {
	n := f.Count
	if limit > 0 && n > limit {
		n = limit
	}
	for i := uint32(0); i < n; i++ {
		print(f, i, order)
	}
	if limit > 0 && f.Count > limit {
		fmt.Print("...")
	}
	fmt.Println()
}

// Print a field's name, type, array size, and values up to a given
// limit (or 0 for no limit).  Names are taken from a map, so that it
// can work on private IFDs as long as they use the standard TIFF data
// types.
func (f Field) Print(order binary.ByteOrder, tagNames map[Tag]string, limit uint32) {
	tagName, found := tagNames[f.Tag]
	if found {
		fmt.Printf("%s %s(%d)", tagName, f.Type.Name(), f.Count)
	} else {
		fmt.Printf("Unknown %d(0x%X) %s(%d)", f.Tag, f.Tag, f.Type.Name(), f.Count)
	}
	switch {
	case f.Type == ASCII:
		str := f.ASCII()
		if limit > 0 && len(str) > int(limit) {
			fmt.Printf(" %q...\n", str[:limit])
		} else {
			fmt.Printf(" %q\n", str)
		}
	case f.Type.IsRational():
		ratPrinter := func(f Field, i uint32, order binary.ByteOrder) {
			num, denom := f.AnyRational(i, order)
			fmt.Printf(" %d/%d", num, denom)
		}
		printValues(f, order, limit, ratPrinter)
	case f.Type.IsIntegral():
		intPrinter := func(f Field, i uint32, order binary.ByteOrder) {
			fmt.Printf(" %d", f.AnyInteger(i, order))
		}
		printValues(f, order, limit, intPrinter)
	case f.Type == UNDEFINED:
		undefPrinter := func(f Field, i uint32, order binary.ByteOrder) {
			fmt.Printf(" %X", f.Data[i])
		}
		printValues(f, order, limit, undefPrinter)
	case f.Type.IsFloat():
		floatPrinter := func(f Field, i uint32, order binary.ByteOrder) {
			fmt.Printf(" %e", f.AnyFloat(i, order))
		}
		printValues(f, order, limit, floatPrinter)
	default:
		fmt.Println(" unknown data type")
	}
}

// Slice pointing to a single segment of image data.
type ImageSegment []byte

// IDs of fields that specify image data. One field has an array of offsets
// and the other an array of sizes, e.g., StripOffsets and StripByteCounts
// in TIFF IFDs. SizeField can be zero in special cases where it's not
// used.
type ImageDataSpec struct {
	OffsetTag Tag
	SizeTag   Tag
}

// Image data segments for a single pair of fields (offsets and sizes).
type ImageData struct {
	OffsetTag Tag
	SizeTag   Tag
	Segments  []ImageSegment
}

// Fields and image data for a single IFD.
type IFD_T struct {
	// Usually, all IFDs in a TIFF file have the same byte order,
	// specified at the start of the file. Certain maker notes use
	// a fixed order instead.
	Order     binary.ByteOrder
	Fields    []Field
	ImageData []ImageData
}

// Image data specifications for TIFF IFDs.

var StripImageData = ImageDataSpec{StripOffsets, StripByteCounts}
var TileImageData = ImageDataSpec{TileOffsets, TileByteCounts}
var FreeImageData = ImageDataSpec{FreeOffsets, FreeByteCounts}

// Single data block, but should work as normal.
var JPEGInterchangeImageData = ImageDataSpec{JPEGInterchangeFormat, JPEGInterchangeFormatLength}

// Obsolete JPEG fields are special cases.
var JPEGQTablesImageData = ImageDataSpec{JPEGQTables, 0}
var JPEGDCTablesImageData = ImageDataSpec{JPEGDCTables, 0}
var JPEGACTablesImageData = ImageDataSpec{JPEGACTables, 0}

var TIFFImageData = []ImageDataSpec{StripImageData, TileImageData, FreeImageData, JPEGInterchangeImageData, JPEGQTablesImageData, JPEGDCTablesImageData, JPEGACTablesImageData}

// Return the size of an IFD if it was serialized, not including any exernal
// data or extra maker note headers.
func (ifd IFD_T) Size() uint32 {
	// 2 bytes for the entry count, 12 for each entry, and 4 for the
	// position of the next ifd.
	return 2 + uint32(len(ifd.Fields))*12 + 4
}

// Return pointers to fields in the IFD that match the given tags. The
// number of returned fields may be less than the number of tags, if
// not all tags are found, or greater if there are duplicate tags
// (which is probably not valid).
func (ifd IFD_T) FindFields(tags []Tag) []*Field {
	fields := make([]*Field, 0, len(tags))
	for i := range ifd.Fields {
		for _, tag := range tags {
			if ifd.Fields[i].Tag == tag {
				fields = append(fields, &ifd.Fields[i])
			}
		}
	}
	return fields
}

// The size of a TIFF header.
// byte order (2 bytes), magic number (2 bytes), IFD position (4 bytes)
const HeaderSize = 8

// Try to read a TIFF header from a slice. Returns an indication of
// validity, the byte order, and the position of the 0th IFD.
func GetHeader(buf []byte) (bool, binary.ByteOrder, uint32) {
	pos := uint32(0)
	var order binary.ByteOrder
	if buf[pos] == 0x49 && buf[pos+1] == 0x49 {
		order = binary.LittleEndian
	} else if buf[pos] == 0x4d && buf[pos+1] == 0x4d {
		order = binary.BigEndian
	} else {
		return false, order, 0
	}
	pos += 2
	if order.Uint16(buf[pos:]) != 42 {
		return false, order, 0
	}
	pos += 2
	ifdPos := order.Uint32(buf[pos:])
	if ifdPos == 0 {
		// TIFF must contain at least one IFD.
		return false, order, 0
	}
	return true, order, ifdPos
}

// Create a TIFF header at the beginning of a byte slice with given byte
// ordering and position of the 0th IFD. Eight bytes will be used.
func PutHeader(buf []byte, order binary.ByteOrder, ifdPos uint32) {
	if order == binary.LittleEndian {
		buf[0] = 0x49
		buf[1] = 0x49
	} else if order == binary.BigEndian {
		buf[0] = 0x4d
		buf[1] = 0x4d
	} else {
		panic("PutHeader: invalid value of 'order'")
	}
	order.PutUint16(buf[2:], 42)
	order.PutUint32(buf[4:], ifdPos)
}

type imageDataFields struct {
	offsetField *Field
	sizeField   *Field
}

// Build an ImageData structure from a buffer.
func getImageData(buf []byte, specF []imageDataFields, order binary.ByteOrder) ([]ImageData, *GetIFDError) {
	bufsize := uint32(len(buf))
	imageData := make([]ImageData, 0, len(specF))
	for i := 0; i < len(specF); i++ {
		if specF[i].offsetField != nil {
			segments := make([]ImageSegment, specF[i].offsetField.Count)
			for j := uint32(0); j < specF[i].offsetField.Count; j++ {
				var size, offset uint32
				// Old-style JPEG tags have no size fields.
				switch specF[i].offsetField.Tag {
				case JPEGQTables:
					offset = specF[i].offsetField.Long(j, order)
					size = 64
				case JPEGDCTables, JPEGACTables:
					offset = specF[i].offsetField.Long(j, order)
					if offset+15 < offset || offset+15 >= bufsize {
						return nil, &GetIFDError{4, 0, 0, specF[i].offsetField.Tag}
					}
					numvals := uint32(0)
					for k := uint32(0); k < 16; k++ {
						numvals += uint32(buf[offset+k])
					}
					size = 16 + numvals
				default:
					if specF[i].sizeField != nil {
						offset = uint32(specF[i].offsetField.AnyInteger(j, order))
						size = uint32(specF[i].sizeField.AnyInteger(j, order))
					}
				}
				if size > 0 {
					if offset+size-1 < offset || offset+size-1 > bufsize {
						return nil, &GetIFDError{4, 0, 0, specF[i].offsetField.Tag}
					}
					segments[j] = buf[offset : offset+size]
				}
			}
			sizeTag := Tag(0)
			if specF[i].sizeField != nil {
				sizeTag = specF[i].sizeField.Tag
			}
			imageData = append(imageData, ImageData{specF[i].offsetField.Tag, sizeTag, segments})
		}
	}
	return imageData, nil
}

const (
	ErrIFDPos       = 1 // IFD position outside buffer.
	ErrIFDTruncated = 2 // IFD doesn't fit in buffer.
	ErrFieldData    = 3 // Field data pointer outside buffer.
	ErrImageData    = 4 // ImageData pointer outside buffer.
)

type GetIFDError struct {
	ErrType    int
	IFDPos     uint32
	IFDEntries uint16
	FieldTag   Tag
}

func (err GetIFDError) Error() string {
	switch err.ErrType {
	case ErrIFDPos:
		return fmt.Sprintf("Attempted to read IFD at position %d, past end of input", err.IFDPos)
	case ErrIFDTruncated:
		return fmt.Sprintf("IFD at offset %d with %d fields extends past end of input", err.IFDPos, err.IFDEntries)
	case ErrFieldData:
		return fmt.Sprintf("When reading IFD at offset %d, data for tag %d extends past end of input", err.IFDPos, err.FieldTag)
	case ErrImageData:
		return fmt.Sprintf("When reading IFD at offset %d, image data for tag %d extends past end of input", err.IFDPos, err.FieldTag)
	default:
		return "Invalid error"
	}
}

// Procedure to be applied to fields after they are read from an
// IFD. Receives field index, field struct, and data position within
// buffer. No return values. Used below to record the position of a
// maker note field.
type FieldProc func(uint16, *Field, uint32)

// Read an IFD and its image data, from a given position in a 'buf'.
// 'spec' specifies the fields that may refer to image data: it will
// probably be TIFFImageData if reading a TIFF IFD or nil if reading a
// private IFD. 'FieldProc' is a procedure for additional field
// processing, or nil if not required. Return values are the IFD, the
// position of the next IFD or 0 if none, and an error status.  Field
// and image data in the returned IFD will point into the slice, so
// modifying one will modify the other. Any error will be of type
// GetIFDError.
func GetIFD(buf []byte, order binary.ByteOrder, pos uint32, spec []ImageDataSpec, fieldProc FieldProc) (IFD_T, uint32, error) {
	var ifd IFD_T
	ifd.Order = order
	ifdpos := pos
	bufsize := uint32(len(buf))
	if pos+2 < pos || pos+2 > bufsize {
		return ifd, 0, GetIFDError{ErrIFDPos, ifdpos, 0, 0}
	}
	entries := order.Uint16(buf[pos:]) // IFD entry count.
	if entries == 0 {
		// May be technically invalid, but just return an IFD with
		// no entries.
		return ifd, 0, nil
	}
	fields := make([]Field, entries)
	ifd.Fields = fields
	if pos+ifd.Size() < pos || pos+ifd.Size() > bufsize {
		ifd.Fields = nil
		return ifd, 0, GetIFDError{ErrIFDTruncated, ifdpos, entries, 0}
	}
	pos += 2
	specF := make([]imageDataFields, len(spec))
	for i := uint16(0); i < entries; i++ {
		fields[i].Tag = Tag(order.Uint16(buf[pos:]))
		pos += 2
		fields[i].Type = Type(order.Uint16(buf[pos:]))
		pos += 2
		fields[i].Count = order.Uint32(buf[pos:])
		pos += 4
		size := fields[i].Size()
		dataPos := pos
		if size > 4 {
			dataPos = order.Uint32(buf[pos:])
			if dataPos+size-1 < dataPos || dataPos+size-1 > bufsize {
				return ifd, 0, GetIFDError{ErrFieldData, ifdpos, entries, fields[i].Tag}
			}
		}
		fields[i].Data = buf[dataPos : dataPos+size]
		if fieldProc != nil {
			fieldProc(i, &fields[i], dataPos)
		}
		pos += 4
		for j := range spec {
			if spec[j].OffsetTag == fields[i].Tag {
				specF[j].offsetField = &fields[i]
			}
			if spec[j].SizeTag == fields[i].Tag {
				specF[j].sizeField = &fields[i]
			}
		}
	}
	var err *GetIFDError
	ifd.ImageData, err = getImageData(buf, specF, order)
	if err != nil {
		err.IFDPos = ifdpos
		err.IFDEntries = entries
		return ifd, 0, err
	}
	next := order.Uint32(buf[pos:])
	return ifd, next, nil
}

// Align a position to the next word (2 byte) boundary.
func Align(pos uint32) uint32 {
	if pos/2*2 != pos {
		return pos + 1
	}
	return pos
}

// Specify the serialized position of a SubIFD.
type IFDpos struct {
	Tag  Tag // field that refers to the subIFD.
	Pos  uint32
	Size uint32 // for maker notes only.
}

// Put image data into buffer at pos. Return next data position in buf and
// a mapping from field tag to an array of offsets where image data was
// placed.
func putImageData(buf []byte, ifd IFD_T, pos uint32, order binary.ByteOrder) (uint32, map[Tag][]byte, error) {
	offsetTags := make([]Tag, len(ifd.ImageData))
	for i := range ifd.ImageData {
		offsetTags[i] = ifd.ImageData[i].OffsetTag
	}
	offsetFields := ifd.FindFields(offsetTags)
	if len(offsetFields) != len(offsetTags) {
		return pos, nil, errors.New("putImageData: ImageData offset fields don't match IFD")
	}
	offsetMap := make(map[Tag][]byte)
	for i, id := range ifd.ImageData {
		offsetData := make([]byte, offsetFields[i].Size())
		offsetMap[offsetTags[i]] = offsetData
		if offsetFields[i].Type != LONG && offsetFields[i].Type != SHORT {
			return pos, offsetMap, errors.New("putImageData: OffsetField not LONG or SHORT")
		}
		for j, seg := range id.Segments {
			copy(buf[pos:], seg)
			if offsetFields[i].Type == LONG {
				order.PutUint32(offsetData[j*4:], pos)
			} else {
				// Rewriting a file may fail if an offset
				// is in a SHORT field and we are trying to
				// write it too high. The solution is
				// probably to convert such fields to LONG
				// before encoding, IFD_T.Fix().
				if pos >= 2<<15 {
					return pos, offsetMap, errors.New("putImageData: position is too large for SHORT field")
				}
				order.PutUint16(offsetData[j*2:], uint16(pos))
			}
			pos += uint32(len(seg))
		}
	}
	return pos, offsetMap, nil
}

// Serialize an IFD and its external data into a byte slice at 'pos'.
// Returns the position following the last byte used. 'buf' must
// represent a serialized TIFF file with the start of the file at the
// start of the slice, and it must be sufficiently large for the new
// data. 'pos' must be word (2 byte) aligned and the tags in the
// fields must be in assending order, according to the TIFF
// specification. 'subifds' supplies the positions of any subIFDs
// refered to by fields in this IFD. 'next' supplies the position of
// the next IFD, or 0 if none.
func (ifd IFD_T) Put(buf []byte, pos uint32, subifds []IFDpos, nextptr uint32) (uint32, error) {
	order := ifd.Order
	if pos/2*2 != pos {
		return 0, errors.New("IFD_T.Put: pos is not word aligned")
	}
	datapos := pos + ifd.Size()
	// Order in the buffer will be 1) IFD 2) image data 3) IFD external data
	datapos, offsets, err := putImageData(buf, ifd, datapos, order)
	if err != nil {
		return 0, err
	}
	numFields := len(ifd.Fields)
	order.PutUint16(buf[pos:], uint16(numFields))
	pos += 2
	var lastTag Tag
	var subifdPtrs = make([]*IFDpos, 0, len(subifds))
	for _, field := range ifd.Fields {
		if field.Tag < lastTag {
			return 0, errors.New(fmt.Sprintf("IFD_T.Put: tags are out of order, %d(0x%X) is followed by %d(0x%X)", lastTag, lastTag, field.Tag, field.Tag))
		}
		lastTag = field.Tag
		order.PutUint16(buf[pos:], uint16(field.Tag))
		pos += 2
		order.PutUint16(buf[pos:], uint16(field.Type))
		pos += 2
		// We can handle two kinds of subIFDs. Firstly, fields
		// that just contain a pointer to one or more subIFDs,
		// usually in a field with LONG or IFD type. Secondly,
		// fields that are arrays of UNDEFINED values, e.g.,
		// maker notes.
		subifdPtrs = subifdPtrs[0:0]
		for i := range subifds {
			if subifds[i].Tag == field.Tag {
				subifdPtrs = append(subifdPtrs, &subifds[i])
			}
		}
		if len(subifdPtrs) > 0 && field.Type == UNDEFINED {
			if len(subifdPtrs) > 1 {
				errors.New("IFD_T.Put: UNDEFINED field expected to have a single IFD.")
			}
			order.PutUint32(buf[pos:], subifdPtrs[0].Size)
			pos += 4
			order.PutUint32(buf[pos:], subifdPtrs[0].Pos)
			pos += 4
			continue
		}
		order.PutUint32(buf[pos:], field.Count)
		pos += 4
		data := field.Data
		size := field.Size()
		if len(subifdPtrs) > 0 {
			// Field points to one or more sub-IFDs.
			if field.Type.Size() != 4 {
				return 0, errors.New("IFD_T.Put: sub-IFD pointer expected to have field type with size 4")
			}
			data = make([]byte, size)
			for i := range subifdPtrs {
				order.PutUint32(data[i*4:], subifdPtrs[i].Pos)
			}
		} else {
			imagedata := offsets[field.Tag]
			if imagedata != nil {
				// Image data offset field.
				data = imagedata
			}
		}
		if size <= 4 {
			copy(buf[pos:], "\000\000\000\000")
			copy(buf[pos:], data[0:size])
		} else {
			order.PutUint32(buf[pos:], datapos)
			copy(buf[datapos:datapos+size], data)
			datapos += size
		}
		pos += 4
	}
	order.PutUint32(buf[pos:], nextptr)
	return datapos, nil
}

// Add some fields to an IFD.
func (ifd *IFD_T) AddFields(fields []Field) {
	addLen := len(fields)
	if addLen == 0 {
		return
	}
	curLen := len(ifd.Fields)
	newLen := curLen + addLen
	var newFields []Field
	if cap(ifd.Fields) < newLen {
		newFields = make([]Field, newLen)
		copy(newFields, ifd.Fields)
	} else {
		newFields = ifd.Fields[:curLen+addLen]
	}
	for i := 0; i < addLen; i++ {
		newFields[curLen+i] = fields[i]
	}
	sort.Slice(newFields, func(i, j int) bool { return newFields[i].Tag < newFields[j].Tag })
	ifd.Fields = newFields
}

// Delete some fields from an IFD.
func (ifd *IFD_T) DeleteFields(tags []Tag) {
	shift := 0
	numFields := len(ifd.Fields)
	for i := 0; i < numFields; i++ {
		if shift > 0 {
			ifd.Fields[i-shift] = ifd.Fields[i]
		}
		for _, t := range tags {
			if ifd.Fields[i].Tag == t {
				shift++
			}
		}
	}
	ifd.Fields = ifd.Fields[:numFields-shift]
}

// IFD Tag namespace.
type TagSpace uint8

// Some information about private IFDs is neded to successfully decode
// TIFF files that use them, since they use the LONG data type instead
// of the IFD data type.
const (
	TIFFSpace         TagSpace = 0
	UnknownSpace      TagSpace = 1
	ExifSpace         TagSpace = 2
	GPSSpace          TagSpace = 3
	InteropSpace      TagSpace = 4
	MPFIndexSpace     TagSpace = 5 // Multi-Picture Format.
	MPFAttributeSpace TagSpace = 6
	// Maker notes below. If adding another, uses of
	// Panasonic1Space in this file will indicate where support is
	// needed.
	Nikon1Space        TagSpace = 7
	Nikon2Space        TagSpace = 8
	Nikon2PreviewSpace TagSpace = 9
	Nikon2ScanSpace    TagSpace = 10
	Panasonic1Space    TagSpace = 11
)

// Return the name of a tag namespace.
func (space TagSpace) Name() string {
	switch space {
	case TIFFSpace:
		return "TIFF"
	case ExifSpace:
		return "Exif"
	case GPSSpace:
		return "GPS"
	case InteropSpace:
		return "Interop"
	case MPFIndexSpace:
		return "MPFIndex"
	case MPFAttributeSpace:
		return "MPFAttribute"
	case Nikon1Space:
		return "Nikon1"
	case Nikon2Space:
		return "Nikon2"
	case Nikon2PreviewSpace:
		return "Nikon2Preview"
	case Nikon2ScanSpace:
		return "Nikon2Scan"
	case Panasonic1Space:
		return "Panasonic1"
	case UnknownSpace:
		return "Unknown"
	}
	panic("TagSpace.Name: invalid value")
}

// Return the byte order for an IFD with given tag namespace, given a
// default order for a TIFF IFD tree. It will usually be the same as the
// default, but may differ for certain maker note IFDs.
func (space TagSpace) ByteOrder(deforder binary.ByteOrder) binary.ByteOrder {
	return deforder
}

// Given a field 'tag' in a 'space' IFD which refers to a sub-IFD,
// return the name space of the sub-IFD.
func (space TagSpace) SubSpace(tag Tag) TagSpace {
	switch space {
	case TIFFSpace:
		switch tag {
		case SubIFDs:
			return TIFFSpace
		case ExifIFD:
			return ExifSpace
		case GPSIFD:
			return GPSSpace
		}
	case ExifSpace:
		if tag == interOpIFD {
			return InteropSpace
		}
	case Nikon2Space:
		if tag == Nikon2PreviewIFD {
			return Nikon2PreviewSpace
		} else if tag == Nikon2NikonScanIFD {
			return Nikon2ScanSpace
		}
	}
	return UnknownSpace
}

// A interface for node-space-specific functionality. Most nodes will
// use the TIFF standard, but maker notes could need anything.
type SpaceRec interface {
	GetSpace() TagSpace
	IsMakerNote() bool
	Size(IFDNode) uint32
	PutIFDTree(IFDNode, []byte, uint32) (uint32, error)
}

// SpaceRec for standard TIFF IFD trees.
type TIFFSpaceRec struct {
	Space TagSpace
}

func (rec TIFFSpaceRec) GetSpace() TagSpace {
	return rec.Space
}

func (TIFFSpaceRec) IsMakerNote() bool {
	return false
}

func (TIFFSpaceRec) Size(node IFDNode) uint32 {
	return node.sizeTIFF()
}

func (TIFFSpaceRec) PutIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.putIFDTreeTIFF(buf, pos)
}

// SpaceRec for Nikon1 maker notes.
type Nikon1SpaceRec struct {
}

func (Nikon1SpaceRec) GetSpace() TagSpace {
	return Nikon1Space
}

func (Nikon1SpaceRec) IsMakerNote() bool {
	return true
}

var nikon1Label = []byte("Nikon\000\001\000")

func (Nikon1SpaceRec) Size(node IFDNode) uint32 {
	return uint32(len(nikon1Label)) + node.sizeTIFF()
}

func (Nikon1SpaceRec) PutIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	copy(buf[pos:], nikon1Label)
	pos += uint32(len(nikon1Label))
	return node.putIFDTreeTIFF(buf, pos)
}

// SpaceRec for Nikon2 maker notes.
type Nikon2SpaceRec struct {
	// The maker note header/label varies, but the tags are
	// compatible. Model examples:
	// Coolpix 990: no header
	// Coolpix 5000: "Nikon\0\2\0\0\0"
	// ??? : "\0\2\x10\0\0"
	// Nikon D500: "Nikon\0\2\x11\0\0"
	label []byte
}

func (Nikon2SpaceRec) GetSpace() TagSpace {
	return Nikon2Space
}

func (Nikon2SpaceRec) IsMakerNote() bool {
	return true
}

var nikon2LabelPrefix = []byte("Nikon\000")

func (rec Nikon2SpaceRec) Size(node IFDNode) uint32 {
	labelLen := len(rec.label)
	if labelLen == 0 {
		// maker note without label or TIFF header.
		return node.sizeTIFF()
	}
	return uint32(labelLen) + HeaderSize + node.sizeTIFF()
}

func (rec Nikon2SpaceRec) PutIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	if len(rec.label) == 0 {
		// maker note without label or TIFF header.
		return node.putIFDTreeTIFF(buf, pos)
	}
	copy(buf[pos:], rec.label)
	pos += uint32(len(rec.label))
	makerBuf := buf[pos:]
	PutHeader(makerBuf, node.IFD_T.Order, HeaderSize)
	last, err := node.putIFDTreeTIFF(makerBuf, HeaderSize)
	if err != nil {
		return 0, err
	}
	return pos + last, nil
}

// Fields in Nikon2 IFDs that may affect IFD structure.
const Nikon2PreviewIFD = 0x11
const Nikon2NikonScanIFD = 0xE10
const Nikon2MakerNoteVersion = 0x1

// SpaceRec for Panasonic1 maker notes.
type Panasonic1SpaceRec struct {
}

func (Panasonic1SpaceRec) GetSpace() TagSpace {
	return Panasonic1Space
}

func (Panasonic1SpaceRec) IsMakerNote() bool {
	return true
}

var panasonicLabel = []byte("Panasonic\000\000\000")

func (Panasonic1SpaceRec) Size(node IFDNode) uint32 {
	return uint32(len(panasonicLabel)) + node.sizeTIFF()
}

func (Panasonic1SpaceRec) PutIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	copy(buf[pos:], panasonicLabel)
	pos += uint32(len(panasonicLabel))
	return node.putIFDTreeTIFF(buf, pos)
}

// TIFF IFD with links to subifds referred to from any field, and to the
// next IFD, if any.
type IFDNode struct {
	IFD_T
	SpaceRec
	SubIFDs []SubIFD
	Next    *IFDNode
}

// TIFF subifd and the field in the parent that referred to it.
type SubIFD struct {
	Tag  Tag
	Node *IFDNode
}

// Create a new TIFF IFDNode with a given name space (not for maker notes).
func NewIFDNodeTIFF(space TagSpace) *IFDNode {
	return &IFDNode{SpaceRec:TIFFSpaceRec{space}}
}

// Fields in Exif IFDs that affect IFD structure.
const interOpIFD = 0xA005
const makerNote = 0x927C

// Data needed for unpacking Exif maker notes.
type makerNoteData struct {
	make     string
	model    string
	position uint32
	exifNode *IFDNode
}

// Helper for GetIFDTree. ifdPositions records byte positions of known
// IFDs so that loops can be detected.
func getIFDTreeIter(buf []byte, order binary.ByteOrder, pos uint32, space TagSpace, maker *makerNoteData, ifdPositions map[uint32]bool) (*IFDNode, error) {
	var node IFDNode
	node.SpaceRec = &TIFFSpaceRec{space}
	var specs []ImageDataSpec
	if space == TIFFSpace {
		specs = TIFFImageData
	}
	if ifdPositions[pos] {
		return nil, errors.New("GetIFDTreeIter: IFD reference loop detected")
	}
	ifdPositions[pos] = true
	var next uint32
	var err error
	var makerNotePos uint32
	var fieldProc FieldProc
	if space == ExifSpace {
		fieldProc = func(fieldNo uint16, field *Field, dataPos uint32) {
			if field.Tag == makerNote {
				makerNotePos = dataPos
			}
		}
	}
	node.IFD_T, next, err = GetIFD(buf, order, pos, specs, fieldProc)
	if err != nil {
		return nil, err
	}
	subnum := uint32(0)
	node.SubIFDs = make([]SubIFD, 0, 10)
	for i, field := range node.Fields {
		if field.IsIFD(space) {
			// Generally, a field references a single IFD, but
			// SubIFDs can point to multiple IFDs.
			// Maker notes aren't processed here.
			for j := uint32(0); j < field.Count; j++ {
				subspace := space.SubSpace(field.Tag)
				var sub SubIFD
				sub.Tag = node.Fields[i].Tag
				sub.Node, err = getIFDTreeIter(buf, order, field.Long(j, order), subspace, maker, ifdPositions)
				if err != nil {
					return nil, err
				}
				node.SubIFDs = append(node.SubIFDs, sub)
				subnum++
			}
		} else {
			if space == TIFFSpace {
				if field.Tag == Make {
					maker.make = field.ASCII()
				} else if field.Tag == Model {
					maker.model = field.ASCII()
				}
			} else if space == ExifSpace {
				if field.Tag == makerNote {
					maker.position = makerNotePos
					maker.exifNode = &node
				}
			}
		}
	}
	if next != 0 {
		var nextnode IFDNode
		node.Next = &nextnode
		var nextSpace TagSpace
		if space == ExifSpace {
			// The next IFD after an Exif IFD is a thumbnail
			// encoded as TIFF.
			nextSpace = TIFFSpace
		} else if space == MPFIndexSpace {
			nextSpace = MPFAttributeSpace
		} else {
			// Assume the next IFD is the same type.
			nextSpace = space
		}
		node.Next, err = getIFDTreeIter(buf, order, next, nextSpace, maker, ifdPositions)
		if err != nil {
			return nil, err
		}
	}
	return &node, nil
}

// Unpack maker note data, and append its IFD to SubIFDs in the Exif node.
func getMakerNote(buf []byte, order binary.ByteOrder, maker makerNoteData) error {
	var makerNode *IFDNode
	switch {
	case bytes.HasPrefix(buf[maker.position:], nikon1Label):
		var node IFDNode
		var err error
		node.IFD_T, _, err = GetIFD(buf, order, maker.position+uint32(len(nikon1Label)), nil, nil)
		if err != nil {
			return err
		}
		node.SpaceRec = &Nikon1SpaceRec{}
		makerNode = &node
	case bytes.HasPrefix(buf[maker.position:], nikon2LabelPrefix):
		label := buf[maker.position : int(maker.position)+len(nikon2LabelPrefix)+4]
		// Contains a new TIFF block with relative offsets.
		tiff := buf[maker.position+uint32(len(label)):]
		valid, order, pos := GetHeader(tiff)
		if !valid {
			return errors.New("TIFF header not found in Nikon2 maker note")
		}
		node, err := GetIFDTree(tiff, order, pos, Nikon2Space)
		if err != nil {
			return err
		}
		node.SpaceRec = &Nikon2SpaceRec{label}
		makerNode = node
	case bytes.HasPrefix(buf[maker.position:], panasonicLabel):
		var node IFDNode
		var err error
		node.IFD_T, _, err = GetIFD(buf, order, maker.position+uint32(len(panasonicLabel)), nil, nil)
		if err != nil {
			return err
		}
		node.SpaceRec = &Panasonic1SpaceRec{}
		makerNode = &node

		// Didn't recognize any maker note label above. Assume the
		// maker note is appropriate for the camera make and/or model.

	case strings.HasPrefix(strings.ToLower(maker.make), "nikon"):
		// Unlabelled maker notes can be produced by early cameras like
		// Coolpix 775 and 990. Like Nikon1 maker notes above, there's
		// no separate TIFF header, but they use the Nikon2 tags.
		// Don't assume the byte order is the same as the Exif block,
		// but the number of tags should be less than 255.
		var order binary.ByteOrder
		if buf[maker.position] == 0 {
			order = binary.BigEndian
		} else {
			order = binary.LittleEndian
		}
		node, err := GetIFDTree(buf, order, maker.position, Nikon2Space)
		if err != nil {
			return err
		}
		node.SpaceRec = &Nikon2SpaceRec{nil}
		// Check if it seems to be a Nikon2 maker note.
		fields := node.FindFields([]Tag{Nikon2MakerNoteVersion})
		if len(fields) == 1 && fields[0].Type == UNDEFINED && fields[0].Count == 4 {
			makerNode = node
		}
	}
	if makerNode != nil {
		var sub SubIFD
		sub.Tag = makerNote
		sub.Node = makerNode
		maker.exifNode.SubIFDs = append(maker.exifNode.SubIFDs, sub)
	}
	return nil
}

// Create an IFDNode tree by reading an IFD and all the other IFDs to
// which it refers. 'pos' is the position of the root IFD in the byte
// slice. 'space' is the name space to assign to the root, usually
// TIFFSpace.
func GetIFDTree(buf []byte, order binary.ByteOrder, pos uint32, space TagSpace) (*IFDNode, error) {
	ifdPositions := make(map[uint32]bool)
	var maker makerNoteData
	tree, err := getIFDTreeIter(buf, order, pos, space, &maker, ifdPositions)
	if err != nil {
		return tree, err
	}
	// Unpacking maker notes requires data from the TIFF and Exif
	// IFDs, so is done now after everything else has been unpacked.
	if maker.position != 0 {
		if err := getMakerNote(buf, order, maker); err != nil {
			return tree, err
		}
	}
	return tree, err
}

// Return the serialized size of a node, including its IFD, external data,
// image data, and maker note headers, but excluding other nodes to which
// it refers.
func (node IFDNode) Size() uint32 {
	return node.SpaceRec.Size(node)
}

// Calculation of serialized size for standard TIFF nodes.
func (node IFDNode) sizeTIFF() uint32 {
	size := node.IFD_T.Size()
FIELDLOOP:
	for _, field := range node.Fields {
		// Don't double-count arrays that have been unpacked
		// into subIFDs (such as maker notes). Assume that any
		// subIFD field with a single-byte type is such an array.
		for i := 0; i < len(node.SubIFDs); i++ {
			if node.SubIFDs[i].Tag == field.Tag && field.Type.Size() == 1 {
				continue FIELDLOOP
			}
		}
		fsize := field.Size()
		if fsize > 4 {
			size += fsize
		}
	}
	for _, id := range node.ImageData {
		for _, seg := range id.Segments {
			size += uint32(len(seg))
		}
	}
	return size
}

// Return the serialized size of a node and all the nodes to which it refers.
// Includes all external data, image data, and maker note headers.
func (node IFDNode) TreeSize() uint32 {
	size := node.Size()
	for i := 0; i < len(node.SubIFDs); i++ {
		size = Align(size)
		size += node.SubIFDs[i].Node.TreeSize()
	}
	size = Align(size)
	if node.Next != nil {
		size += node.Next.TreeSize()
	}
	return size

}

// Allow the PutIFDTree function to be selected according to the node
//  space. Normal TIFF nodes will call putIFDTreeTIFF below.  Serialize
//  an IFD and all the other IFDs to which it refers into a byte slice
//  at 'pos'.  Returns the position following the last byte
//  used.
func (node IFDNode) PutIFDTree(buf []byte, pos uint32) (uint32, error) {
	return node.SpaceRec.PutIFDTree(node, buf, pos)
}

// Version of PutIFDTree, for regular TIFF nodes (maker notes may
// require special treatment.)  'buf' must represent a serialized TIFF
// file with the start of the file at the start of the slice, and it
// must be sufficiently large for the new data. 'pos' must be word (2
// byte) aligned and the tags in the IFDs must be in ascending order,
// according to the TIFF specification.
func (node IFDNode) putIFDTreeTIFF(buf []byte, pos uint32) (uint32, error) {
	// Write node's IFD at pos. But first write any IFDs that it
	// refers to, recording their positions.
	nsubs := len(node.SubIFDs)
	subpos := make([]IFDpos, nsubs)
	next := pos + node.sizeTIFF()
	var err error
	for i := 0; i < nsubs; i++ {
		next = Align(next)
		subpos[i].Tag = node.SubIFDs[i].Tag
		subpos[i].Pos = next
		nextTmp, err := node.SubIFDs[i].Node.PutIFDTree(buf, next)
		if err != nil {
			return 0, err
		}
		subpos[i].Size = nextTmp - next
		next = nextTmp
	}
	nextPos := uint32(0)
	if node.Next != nil {
		next = Align(next)
		nextPos = next
		next, err = node.Next.PutIFDTree(buf, next)
		if err != nil {
			return 0, err
		}
	}
	_, err = node.IFD_T.Put(buf, pos, subpos, nextPos)
	if err != nil {
		return 0, err
	}
	return next, nil
}

// TIFF fixes: *) Sort the fields into ascending Tag order *) TIFF
// allows a SHORT field to contain a pointer to image data. This can
// fail if we write image data at a different location in the file, so
// convert such fields to LONG. *) Add missing NUL terminators in
// ASCII field data. Additional fixes may be added later.
func (ifd *IFD_T) Fix(specs []ImageDataSpec) {
	sort.Slice(ifd.Fields, func(i, j int) bool { return ifd.Fields[i].Tag < ifd.Fields[j].Tag })
	for i := 0; i < len(ifd.Fields); i++ {
		field := &ifd.Fields[i]
		if field.Type == SHORT {
			for j := range specs {
				if specs[j].OffsetTag == field.Tag {
					offsets := make([]uint32, field.Count)
					for k := uint32(0); k < field.Count; k++ {
						offsets[k] = uint32(field.Short(k, ifd.Order))
					}
					field.Type = LONG
					field.Data = make([]byte, 4*field.Count)
					for k := uint32(0); k < field.Count; k++ {
						field.PutLong(offsets[k], k, ifd.Order)
					}
					break
				}
			}
		} else if field.Type == ASCII {
			if field.Data[field.Count-1] != 0 {
				field.Count++
				newData := make([]byte, field.Count)
				copy(newData, field.Data)
				field.Data = newData
			}
		}
	}
}

// Apply IFD fixes to all IFDs in a tree.
func (node *IFDNode) Fix() {
	var specs []ImageDataSpec
	if node.GetSpace() == TIFFSpace {
		specs = TIFFImageData
	}
	node.IFD_T.Fix(specs)
	for i := 0; i < len(node.SubIFDs); i++ {
		node.SubIFDs[i].Node.Fix()
	}
	if node.Next != nil {
		node.Next.Fix()
	}
}
