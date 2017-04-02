package tiff66

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
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
	XMP                         = 0x2BC // XMP part 3
	ImageID                     = 0x800 // Supplement 1
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

// Mapping to names of tags lists above.
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

// Indicate if this field a pointer to another IFD. This works on TIFF IFDs,
// not necessarily on private ones.
func (f Field) IsIFD() bool {
	return f.Type == IFD || f.Tag == SubIFDs || f.Tag == ExifIFD || f.Tag == GPSIFD
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
		fmt.Printf(" %q\n", f.ASCII())
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
// in TIFF IFDs.
type ImageDataSpec struct {
	OffsetTag Tag
	SizeTag   Tag
}

// Image data segments for a single pair of fields (offsets and sizes).
type ImageData struct {
	OffsetField *Field
	SizeField   *Field
	Segments    []ImageSegment
}

// Fields and image data for a single IFD.
type IFD_T struct {
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
// data.
func (ifd IFD_T) Size() uint32 {
	// 2 bytes for the entry count, 12 for each entry, and 4 for the
	// position of the next ifd.
	return 2 + uint32(len(ifd.Fields))*12 + 4
}

// Return the size of the external data of a TIFF IFD if was serialized.
// Includes field data and image data, but not sub IFDs.
func (ifd IFD_T) DataSize(order binary.ByteOrder) uint32 {
	var datasize uint32
	for _, field := range ifd.Fields {
		size := field.Size()
		if size > 4 {
			datasize += size
		}
	}
	for _, id := range ifd.ImageData {
		for _, seg := range id.Segments {
			datasize += uint32(len(seg))
		}
	}
	return datasize
}

// Return the size of an IFD and its external data if it was serialized.
// External data includes field data and image data, but not sub IFDs.
func (ifd IFD_T) TotalSize(order binary.ByteOrder) uint32 {
	return ifd.Size() + ifd.DataSize(order)
}

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
func getImageData(buf []byte, specF []imageDataFields, order binary.ByteOrder) []ImageData {
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
					segments[j] = buf[offset : offset+size]
				}
			}
			imageData = append(imageData, ImageData{specF[i].offsetField, specF[i].sizeField, segments})
		}
	}
	return imageData
}

// Implementaion of GetIFD. Invalid TIFF files may cause panicking with
// "slice bounds out of range".
func getIFDImpl(buf []byte, order binary.ByteOrder, pos uint32, spec []ImageDataSpec) (IFD_T, uint32) {
	entries := order.Uint16(buf[pos:]) // IFD entry count.
	pos += 2
	fields := make([]Field, entries)
	specF := make([]imageDataFields, len(spec))
	for i := uint16(0); i < entries; i++ {
		fields[i].Tag = Tag(order.Uint16(buf[pos:]))
		for j := range spec {
			if spec[j].OffsetTag == fields[i].Tag {
				specF[j].offsetField = &fields[i]
			}
			if spec[j].SizeTag == fields[i].Tag {
				specF[j].sizeField = &fields[i]
			}
		}
		pos += 2
		fields[i].Type = Type(order.Uint16(buf[pos:]))
		pos += 2
		fields[i].Count = order.Uint32(buf[pos:])
		pos += 4
		size := fields[i].Size()
		if size <= 4 {
			fields[i].Data = buf[pos : pos+size]
		} else {
			dataPos := order.Uint32(buf[pos:])
			fields[i].Data = buf[dataPos : dataPos+size]
		}
		pos += 4
	}
	var ifd IFD_T
	ifd.Fields = fields
	ifd.ImageData = getImageData(buf, specF, order)
	next := order.Uint32(buf[pos:])
	return ifd, next
}

// Read an IFD and its image data, starting at a given position in a
// byte slice. 'spec' specifies the fields that may refer to image
// data: it will probably be TIFFImageData if reading a TIFF IFD or
// nil if reading a private IFD. Returns the IFD and the position of
// the next IFD, or 0 if none.  Field and image data in the returned
// IFD will point into the slice, so modifying one will modify the
// other. May return a "slice out of bounds" error if the input is
// invalid.
func GetIFD(buf []byte, order binary.ByteOrder, pos uint32, spec []ImageDataSpec) (ifd IFD_T, next uint32, err error) {
	defer func() {
		if val := recover(); val != nil {
			err = fmt.Errorf("%v", val)
		}
	}()
	ifd, next = getIFDImpl(buf, order, pos, spec)
	return ifd, next, err
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
	Tag Tag // field that refers to the subIFD.
	Pos uint32
}

// Put image data into buffer at pos. Return next data position in buf and
// a mapping of field tag to offset array.
func putImageData(buf []byte, imgData []ImageData, pos uint32, order binary.ByteOrder) (uint32, map[Tag][]byte, error) {
	offsets := make(map[Tag][]byte)
	for _, id := range imgData {
		data := make([]byte, id.OffsetField.Size())
		offsets[id.OffsetField.Tag] = data
		if id.OffsetField.Type != LONG && id.OffsetField.Type != SHORT {
			return pos, offsets, errors.New("putImageData: OffsetField not LONG or SHORT")
		}
		for j, seg := range id.Segments {
			copy(buf[pos:], seg)
			if id.OffsetField.Type == LONG {
				order.PutUint32(data[j*4:], pos)
			} else {
				// Rewriting a file may fail if an offset
				// is in a SHORT field and we are trying to
				// write it too high. The solution is
				// probably to convert such fields to LONG
				// before encoding, IFD_T.Fix().
				if pos >= 2<<15 {
					return pos, offsets, errors.New("putImageData: position is too large for SHORT field")
				}
				order.PutUint16(data[j*2:], uint16(pos))
			}
			pos += uint32(len(seg))
		}
	}
	return pos, offsets, nil
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
func (ifd IFD_T) Put(buf []byte, order binary.ByteOrder, pos uint32, subifds []IFDpos, nextptr uint32) (uint32, error) {
	if pos/2*2 != pos {
		return 0, errors.New("PutIFD: pos is not word aligned")
	}
	datapos := pos + ifd.Size()
	// Order in the buffer will be 1) IFD 2) image data 3) IFD external data
	datapos, offsets, err := putImageData(buf, ifd.ImageData, datapos, order)
	if err != nil {
		return 0, err
	}
	numFields := len(ifd.Fields)
	order.PutUint16(buf[pos:], uint16(numFields))
	pos += 2
	var lastTag Tag
	for _, field := range ifd.Fields {
		if field.Tag < lastTag {
			return 0, errors.New("PutIFD: tags are out of order")
		}
		lastTag = field.Tag
		order.PutUint16(buf[pos:], uint16(field.Tag))
		pos += 2
		order.PutUint16(buf[pos:], uint16(field.Type))
		pos += 2
		order.PutUint32(buf[pos:], field.Count)
		pos += 4
		size := field.Size()
		data := field.Data
		imagedata := offsets[field.Tag]
		if imagedata != nil {
			data = imagedata
		}
		for _, subifd := range subifds {
			if subifd.Tag == field.Tag {
				data = make([]byte, 4)
				order.PutUint32(data, subifd.Pos)
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

// IFD Tag namespace.
type TagSpace uint8

// Some information about private IFDs is neded to successfully decode
// TIFF files that use them, since they use the LONG data type instead
// of the IFD data type.
const (
	TIFFSpace    TagSpace = 0
	UnknownSpace TagSpace = 1
	ExifSpace    TagSpace = 2
	GPSSpace     TagSpace = 3
	InteropSpace TagSpace = 4
)

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
	case UnknownSpace:
		return "Unknown"
	}
	panic("TagSpace.Name: invalid value")
}

// TIFF IFD with links to subifds referred to from any field, and to the
// next IFD, if any.
type IFDNode struct {
	IFD     IFD_T
	Space   TagSpace
	SubIFDs []SubIFD
	Next    *IFDNode
}

// TIFF subifd and the field in the parent that referred to it.
type SubIFD struct {
	Field *Field
	Node  *IFDNode
}

const interOpIFD = 0xA005

// Return the IFD space of a sub-IFD referred to by field in a TIFF IFD.
func Space(tag Tag) TagSpace {
	switch tag {
	case SubIFDs:
		return TIFFSpace
	case ExifIFD:
		return ExifSpace
	case GPSIFD:
		return GPSSpace
	default:
		return UnknownSpace
	}
}

// Indicate if this field (in an Exif IFD) is a pointer to another IFD.
func ExifIsIFD(f Field) bool {
	return f.Tag == interOpIFD || f.Type == IFD
}

// Return the IFD space of a sub-IFD referred to by field in an Exif IFD.
func ExifTagSpace(tag Tag) TagSpace {
	switch tag {
	case interOpIFD:
		return InteropSpace
	default:
		return UnknownSpace
	}
}

// Helper for GetIFDTree. ifdPositions records byte positions of known
// IFDs so that loops can be detected.
func getIFDTreeIter(buf []byte, order binary.ByteOrder, pos uint32, space TagSpace, ifdPositions map[uint32]bool) (*IFDNode, error) {
	var node IFDNode
	node.Space = space
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
	node.IFD, next, err = GetIFD(buf, order, pos, specs)
	if err != nil {
		return nil, err
	}
	subnum := uint32(0)
	node.SubIFDs = make([]SubIFD, 0, 10)
	for i, field := range node.IFD.Fields {
		var isIFD bool
		switch node.Space {
		case TIFFSpace:
			isIFD = field.IsIFD()
		case ExifSpace:
			isIFD = ExifIsIFD(field)
		default:
			isIFD = field.Type == IFD
		}
		if isIFD {
			// A SubIFDs field can point to multiple IFDs.
			for j := uint32(0); j < field.Count; j++ {
				var sub SubIFD
				sub.Field = &node.IFD.Fields[i]
				var space TagSpace = UnknownSpace
				switch node.Space {
				case TIFFSpace:
					space = Space(field.Tag)
				case ExifSpace:
					space = ExifTagSpace(field.Tag)
				}
				node.SubIFDs = append(node.SubIFDs, sub)
				node.SubIFDs[subnum].Node, err = getIFDTreeIter(buf, order, field.Long(j, order), space, ifdPositions)
				if err != nil {
					return nil, err
				}
				subnum++
			}
		}
	}
	if next != 0 {
		var nextnode IFDNode
		node.Next = &nextnode
		var space TagSpace
		if node.Space == ExifSpace {
			// The next IFD after an Exif IFD is a thumbnail
			// encoded as TIFF.
			space = TIFFSpace
		} else {
			// Assume the next IFD is the same type.
			space = node.Space
		}
		node.Next, err = getIFDTreeIter(buf, order, next, space, ifdPositions)
		if err != nil {
			return nil, err
		}
	}
	return &node, nil
}

// Read an IFD, and all the other IFDs to which it refers, starting
// from a given position in a byte slice.
func GetIFDTree(buf []byte, order binary.ByteOrder, pos uint32, space TagSpace) (*IFDNode, error) {
	ifdPositions := make(map[uint32]bool)
	return getIFDTreeIter(buf, order, pos, space, ifdPositions)
}

// Return the serialized size of a node and all the nodes to which it refers.
func (node IFDNode) TreeSize(order binary.ByteOrder) uint32 {
	size := uint32(0)
	for i := 0; i < len(node.SubIFDs); i++ {
		size += node.SubIFDs[i].Node.TreeSize(order)
	}
	if node.Next != nil {
		size += node.Next.TreeSize(order)
	}
	tsize := node.IFD.TotalSize(order)
	if tsize/2*2 != tsize {
		// Allow for a filler byte for word alignment.
		tsize++
	}
	size += tsize
	return size

}

// Serialize an IFD and all the other IFDs to which it refers into a
// byte slice at 'pos'.  Returns the position following the last byte
// used. 'buf' must represent a serialized TIFF file with the start of
// the file at the start of the slice, and it must be sufficiently
// large for the new data. 'pos' must be word (2 byte) aligned and the
// tags in the IFDs must be in assending order, according to the TIFF
// specification.
func (node IFDNode) PutIFDTree(buf []byte, pos uint32, order binary.ByteOrder) (uint32, error) {
	// Write node's IFD at pos. But first write any IFDs that it
	// refers to, recording their positions.
	nsubs := len(node.SubIFDs)
	subifds := make([]IFDpos, nsubs)
	next := pos + node.IFD.TotalSize(order)
	var err error
	for i := 0; i < nsubs; i++ {
		next = Align(next)
		subifds[i].Tag = node.SubIFDs[i].Field.Tag
		subifds[i].Pos = next
		next, err = node.SubIFDs[i].Node.PutIFDTree(buf, next, order)
		if err != nil {
			return 0, err
		}
	}
	nodepos := uint32(0)
	if node.Next != nil {
		next = Align(next)
		nodepos = next
		next, err = node.Next.PutIFDTree(buf, next, order)
		if err != nil {
			return 0, err
		}
	}
	_, err = node.IFD.Put(buf, order, pos, subifds, nodepos)
	if err != nil {
		return 0, err
	}
	return next, nil
}

// TIFF fixes: 1) TIFF allows a SHORT field to contain a pointer to
// image data. This can fail if we write image data at a different
// location in the file, so convert such fields to LONG. 2) Add
// missing NUL terminators in ASCII field data. Additional fixes
// may be added later.
func (ifd *IFD_T) Fix(order binary.ByteOrder, specs []ImageDataSpec) {
	for i := 0; i < len(ifd.Fields); i++ {
		field := &ifd.Fields[i]
		if field.Type == SHORT {
			for j := range specs {
				if specs[j].OffsetTag == field.Tag {
					offsets := make([]uint32, field.Count)
					for k := uint32(0); k < field.Count; k++ {
						offsets[k] = uint32(field.Short(k, order))
					}
					field.Type = LONG
					field.Data = make([]byte, 4*field.Count)
					for k := uint32(0); k < field.Count; k++ {
						field.PutLong(offsets[k], k, order)
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
func (node *IFDNode) Fix(order binary.ByteOrder) {
	var specs []ImageDataSpec
	if node.Space == TIFFSpace {
		specs = TIFFImageData
	}
	node.IFD.Fix(order, specs)
	for i := 0; i < len(node.SubIFDs); i++ {
		node.SubIFDs[i].Node.Fix(order)
	}
	if node.Next != nil {
		node.Next.Fix(order)
	}
}
