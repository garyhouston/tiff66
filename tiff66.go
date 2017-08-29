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
	l := len(f.Data)
	if l == 0 {
		return ""
	}
	if f.Data[l-1] == 0 {
		return string(f.Data[:l-1])
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
	case f.Type == IFD:
		ifdPrinter := func(f Field, i uint32, order binary.ByteOrder) {
			fmt.Printf(" %X", f.Long(i, order))
		}
		printValues(f, order, limit, ifdPrinter)
	default:
		fmt.Println(" unknown data type")
	}
}

// Slice pointing to a single segment of image data.
type ImageSegment []byte

// Image data segments for a single pair of fields (offsets and sizes).
type ImageData struct {
	OffsetTag Tag
	SizeTag   Tag
	Segments  []ImageSegment
}

// The size of a TIFF header.
// byte order (2 bytes), magic number (2 bytes), IFD position (4 bytes)
const HeaderSize = 8

// Try to read a TIFF header from a slice. Returns an indication of
// validity, the byte order, and the position of the 0th IFD.
func GetHeader(buf []byte) (bool, binary.ByteOrder, uint32) {
	var order binary.ByteOrder
	if len(buf) < HeaderSize {
		return false, order, 0
	}
	pos := uint32(0)
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

// Node in an IFD tree.
type IFDNode struct {
	// Usually all IFDs in a TIFF file have the same byte order,
	// specified at the start of the file, but this may not be
	// the case for maker notes.
	Order     binary.ByteOrder
	Fields    []Field
	SpaceRec
	SubIFDs []SubIFD  // Links to sub-IFD nodes linked by fields.
	Next    *IFDNode  // Tail link to next node.
}

// TIFF subifd and the field in the parent that referred to it.
type SubIFD struct {
	Tag  Tag
	Node *IFDNode
}

// Create a new IFDNode with a given namespace.
func NewIFDNode(space TagSpace) *IFDNode {
	return &IFDNode{SpaceRec: NewSpaceRec(space)}
}

// Return the serialized size of a basic IFD table.
func (node IFDNode) TableSize() uint32 {
	// 2 bytes for the entry count, 12 for each entry, and 4 for the
	// position of the next IFD.
	return 2 + uint32(len(node.Fields))*12 + 4
}

// Serialized size of a node, including its IFD, external data, image
// data, and maker note headers, but excluding other nodes to which it
// refers.
func (node IFDNode) NodeSize() uint32 {
	return node.SpaceRec.nodeSize(node)
}

// Version of NodeSize for generic TIFF nodes.
func (node IFDNode) genericSize() uint32 {
	size := node.TableSize()
FIELDLOOP:
	for _, field := range node.Fields {
		// Don't double-count arrays that have been unpacked
		// into subIFDs (such as maker notes). Assume that any
		// subIFD field with a single-byte type is such an array.
		if field.Type.Size() == 1 {
			for i := 0; i < len(node.SubIFDs); i++ {
				if node.SubIFDs[i].Tag == field.Tag {
					continue FIELDLOOP
				}
			}
		}
		fsize := field.Size()
		if fsize > 4 {
			size += fsize
		}
	}
	imageData := node.GetImageData()
	for _, id := range imageData {
		for _, seg := range id.Segments {
			size += uint32(len(seg))
		}
	}
	return size
}

// Align a position to the next word (2 byte) boundary.
func Align(pos uint32) uint32 {
	if pos/2*2 != pos {
		return pos + 1
	}
	return pos
}

// Return the serialized size of a node and all the nodes to which it refers.
// Includes all external data, image data, and maker note headers.
func (node IFDNode) TreeSize() uint32 {
	size := node.NodeSize()
	for i := 0; i < len(node.SubIFDs); i++ {
		size = Align(size)
		size += node.SubIFDs[i].Node.TreeSize()
	}
	if node.Next != nil {
		size = Align(size)
		size += node.Next.TreeSize()
	}
	return size

}

// Return pointers to fields in the IFD that match the given tags. The
// number of returned fields may be less than the number of tags, if
// not all tags are found, or greater if there are duplicate tags
// (which is probably not valid).
func (node IFDNode) FindFields(tags []Tag) []*Field {
	fields := make([]*Field, 0, len(tags))
	for i := range node.Fields {
		for _, tag := range tags {
			if node.Fields[i].Tag == tag {
				fields = append(fields, &node.Fields[i])
			}
		}
	}
	return fields
}

// Add some fields to an IFD.
func (node *IFDNode) AddFields(fields []Field) {
	addLen := len(fields)
	if addLen == 0 {
		return
	}
	curLen := len(node.Fields)
	newLen := curLen + addLen
	var newFields []Field
	if cap(node.Fields) < newLen {
		newFields = make([]Field, newLen)
		copy(newFields, node.Fields)
	} else {
		newFields = node.Fields[:curLen+addLen]
	}
	for i := 0; i < addLen; i++ {
		newFields[curLen+i] = fields[i]
	}
	sort.Slice(newFields, func(i, j int) bool { return newFields[i].Tag < newFields[j].Tag })
	node.Fields = newFields
}

// Delete some fields from an IFD.
func (node *IFDNode) DeleteFields(tags []Tag) {
	shift := 0
	numFields := len(node.Fields)
	for i := 0; i < numFields; i++ {
		if shift > 0 {
			node.Fields[i-shift] = node.Fields[i]
		}
		for _, t := range tags {
			if node.Fields[i].Tag == t {
				shift++
			}
		}
	}
	node.Fields = node.Fields[:numFields-shift]
}

// Create an IFDNode tree by reading an IFD and all the other IFDs to
// which it refers. 'pos' is the position of the root IFD in the byte
// slice. 'space' is the namespace to assign to the root, usually
// TIFFSpace.
func GetIFDTree(buf []byte, order binary.ByteOrder, pos uint32, space TagSpace) (*IFDNode, error) {
	ifdPositions := make(posMap)
	return getIFDTreeIter(buf, order, pos, NewSpaceRec(space), ifdPositions)
}

// Map and key for cycle detection, by recording the positions of
// known IFDs so that cycles can be detected. Such files would be
// invalid, e.g., an IFD that lists its parent as a subIFD, but going
// into an infinite loop when parsing it is undesirable.  Buffer
// position alone isn't sufficient, since sometimes maker notes are
// parsed from a substring of the original buffer. Using the buffer
// length in the key may give a false cycle detection if two
// substrings of the same length were encountered, but this is
// unlikely since there's usually only a single maker note.
type posMap map[[2]uint32]bool

func posKey(buf []byte, pos uint32) [2]uint32 {
	return [2]uint32{uint32(len(buf)), pos}
}

// Helper for GetIFDTree.
func getIFDTreeIter(buf []byte, order binary.ByteOrder, pos uint32, spaceRec SpaceRec, ifdPositions posMap) (*IFDNode, error) {
	var node IFDNode
	node.Order = order
	node.SpaceRec = spaceRec
	return &node, node.SpaceRec.getIFDTree(&node, buf, pos, ifdPositions)
}

// Version of getIFDTreeIter with no subspace-specific processing.
func (node *IFDNode) genericGetIFDTreeIter(buf []byte, pos uint32, ifdPositions posMap) error {
	if ifdPositions[posKey(buf, pos)] {
		return errors.New("getIFDTreeIter: IFD cycle detected")
	}
	ifdPositions[posKey(buf, pos)] = true
	node.SubIFDs = make([]SubIFD, 0, 10)
	ifdpos := pos
	bufsize := uint32(len(buf))
	if pos+2 < pos || pos+2 > bufsize {
		return GetIFDError{ErrIFDPos, ifdpos, 0, 0}
	}
	order := node.Order
	entries := order.Uint16(buf[pos:]) // IFD entry count.
	if entries == 0 {
		// May be technically invalid, but just leave an IFD with
		// no entries.
		return nil
	}
	fields := make([]Field, entries)
	node.Fields = fields
	tableSize := node.TableSize()
	if pos+tableSize < pos || pos+tableSize > bufsize {
		node.Fields = nil
		return GetIFDError{ErrIFDTruncated, ifdpos, entries, 0}
	}
	pos += 2
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
			if dataPos+size < dataPos || dataPos+size > bufsize {
				return GetIFDError{ErrFieldData, ifdpos, entries, fields[i].Tag}
			}
		}
		fields[i].Data = buf[dataPos : dataPos+size]
		pos += 4
		// Space-specific field processing, including subIFD
		// recursion.
		subIFDs, err := node.SpaceRec.takeField(buf, order, ifdPositions, i, &fields[i], dataPos)
		if err != nil {
			return err
		}
		if subIFDs != nil {
			node.SubIFDs = append(node.SubIFDs, subIFDs...)
		}
	}
	next := uint32(0)
	space := node.GetSpace()
	if space != Panasonic1Space {
		// Panasonic maker note omits the pointer.
		next = order.Uint32(buf[pos:])
	}
	if next > 0 {
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
		var err error
		node.Next, err = getIFDTreeIter(buf, order, next, NewSpaceRec(nextSpace), ifdPositions)
		if err != nil {
			return err
		}
	}
	return nil
}

// IFD Tag namespace.
type TagSpace uint8

const (
	TIFFSpace         TagSpace = 0
	UnknownSpace      TagSpace = 1
	ExifSpace         TagSpace = 2
	GPSSpace          TagSpace = 3
	InteropSpace      TagSpace = 4
	MPFIndexSpace     TagSpace = 5 // Multi-Picture Format.
	MPFAttributeSpace TagSpace = 6
	Canon1Space        TagSpace = 7
	Nikon1Space        TagSpace = 8
	Nikon2Space        TagSpace = 9
	Nikon2PreviewSpace TagSpace = 10
	Nikon2ScanSpace    TagSpace = 11
	Olympus1Space      TagSpace = 12
	Olympus1EquipmentSpace     TagSpace = 13
	Olympus1CameraSettingsSpace     TagSpace = 14
	Olympus1RawDevelopmentSpace     TagSpace = 15
	Olympus1RawDev2Space     TagSpace = 16
	Olympus1ImageProcessingSpace     TagSpace = 17
	Olympus1FocusInfoSpace     TagSpace = 18
	Panasonic1Space    TagSpace = 19
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
	case Canon1Space:
		return "Canon1"
	case Nikon1Space:
		return "Nikon1"
	case Nikon2Space:
		return "Nikon2"
	case Nikon2PreviewSpace:
		return "Nikon2Preview"
	case Nikon2ScanSpace:
		return "Nikon2Scan"
	case Olympus1Space:
		return "Olympus1"
	case Olympus1EquipmentSpace:
		return "Olympus1Equipment"
	case Olympus1CameraSettingsSpace:
		return "Olympus1CameraSettings"
	case Olympus1RawDevelopmentSpace:
		return "Olympus1RawDevelopment"
	case Olympus1RawDev2Space:
		return "Olympus1RawDev2"
	case Olympus1ImageProcessingSpace:
		return "Olympus1ImageProcessing"
	case Olympus1FocusInfoSpace:
		return "Olympus1FocusInfo"
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

// A interface for node-space-specific functionality.
type SpaceRec interface {
	GetSpace() TagSpace
	IsMakerNote() bool
	nodeSize(IFDNode) uint32
	takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error)
	getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error
	putIFDTree(IFDNode, []byte, uint32) (uint32, error)
	// Return ImageData, which can be the arrays of scan data that may be
	// found in TIFF nodes, or any other data that's specified with
	// pointers instead of arrays.
	GetImageData() []ImageData
}

// Allocate a new SpaceRec for given tag space.
func NewSpaceRec(space TagSpace) SpaceRec {
	switch space {
	case TIFFSpace:
		return &TIFFSpaceRec{}
	case ExifSpace:
		return &ExifSpaceRec{}
	case Canon1Space:
		return &Canon1SpaceRec{}
	case Nikon1Space:
		return &Nikon1SpaceRec{}
	case Nikon2Space:
		return &Nikon2SpaceRec{}
	case Nikon2PreviewSpace:
		return &Nikon2PreviewSpaceRec{}
	case Olympus1Space:
		return &Olympus1SpaceRec{}
	case Panasonic1Space:
		return &Panasonic1SpaceRec{}
	default:
		return &GenericSpaceRec{space: space}
	}
}

// Recursively read SubIFDs specified with a given field. Such fields
// contain pointer(s) to the SubIFD location(s).
func recurseSubIFDs(buf []byte, order binary.ByteOrder, ifdPositions posMap, field *Field, spaceRec SpaceRec) ([]SubIFD, error) {
	var subIFDs []SubIFD
	for i := uint32(0); i < field.Count; i++ {
		var sub SubIFD
		sub.Tag = field.Tag
		var err error
		sub.Node, err = getIFDTreeIter(buf, order, field.Long(i, order), spaceRec, ifdPositions)
		if err != nil {
			return nil, err
		}
		subIFDs = append(subIFDs, sub)
	}
	return subIFDs, nil
}

// SpaceRec with no special processing.
type GenericSpaceRec struct {
	space TagSpace
}

func (rec *GenericSpaceRec) GetSpace() TagSpace {
	return rec.space
}

func (*GenericSpaceRec) IsMakerNote() bool {
	return false
}

func (*GenericSpaceRec) nodeSize(node IFDNode) uint32 {
	return node.genericSize()
}

func (rec *GenericSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	// Process a field of type IFD: these declare a subIFD, and
	// can be potentially found in any IFD.  Assume the subIFD has
	// the same space as the current IFD.
	if field.Type == IFD {
		return recurseSubIFDs(buf, order, ifdPositions, field, NewSpaceRec(rec.space))
	}
	return nil, nil
}

func (*GenericSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (*GenericSpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (*GenericSpaceRec) GetImageData() []ImageData {
	return nil
}
	
var tiffOffsetTags = []Tag{StripOffsets, TileOffsets, FreeOffsets, JPEGInterchangeFormat}
var tiffSizeTags = []Tag{StripByteCounts, TileByteCounts, FreeByteCounts, JPEGInterchangeFormatLength}
const tiffNumTags = 4

// SpaceRec for TIFF nodes.
type TIFFSpaceRec struct {
	offsetFields [tiffNumTags]*Field
	sizeFields [tiffNumTags]*Field
	make, model string
	imageData []ImageData
}

func (rec *TIFFSpaceRec) GetSpace() TagSpace {
	return TIFFSpace
}

func (*TIFFSpaceRec) IsMakerNote() bool {
	return false
}

func (*TIFFSpaceRec) nodeSize(node IFDNode) uint32 {
	return node.genericSize()
}

func newImageData(buf []byte, order binary.ByteOrder, offsetField, sizeField *Field) (*ImageData, error) {
	segments := make([]ImageSegment, offsetField.Count)
	for i := uint32(0); i < offsetField.Count; i++ {
		offset := uint32(offsetField.AnyInteger(i, order))
		size := uint32(sizeField.AnyInteger(i, order))
		bufsize := uint32(len(buf))
		if offset+size < offset || offset+size > bufsize {
			return nil, &GetIFDError{4, 0, 0, offsetField.Tag}
		}
		segments[i] = buf[offset : offset+size]
	}
	return &ImageData{offsetField.Tag, sizeField.Tag, segments}, nil
}


// Store image data in the TIFF space rec.
func (rec *TIFFSpaceRec) appendImageData(buf []byte, order binary.ByteOrder, offsetField, sizeField *Field) error {
	imageData, err := newImageData(buf, order, offsetField, sizeField)
	if err != nil {
		return err
	}
	rec.imageData = append(rec.imageData, *imageData)
	return nil
}

func (rec *TIFFSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	// SubIFDs.
	if field.Type == IFD || field.Tag == SubIFDs || field.Tag == ExifIFD || field.Tag == GPSIFD {
		var spaceRec SpaceRec
		if field.Tag == ExifIFD {
			spaceRec = &ExifSpaceRec{make:rec.make, model:rec.model}
		} else if field.Tag == GPSIFD {
			spaceRec = NewSpaceRec(GPSSpace)
		} else {
			spaceRec = NewSpaceRec(TIFFSpace)
		}
		return recurseSubIFDs(buf, order, ifdPositions, field, spaceRec)
	}
	
	// ImageData tags.
	for i := 0; i < tiffNumTags; i++ {
		if field.Tag == tiffOffsetTags[i] {
			rec.offsetFields[i] = field
		} else if field.Tag == tiffSizeTags[i] {
			rec.sizeFields[i] = field
		}
		if rec.offsetFields[i] != nil && rec.sizeFields[i] != nil {
			rec.appendImageData(buf, order, rec.offsetFields[i], rec.sizeFields[i])
			rec.offsetFields[i] = nil
			rec.sizeFields[i] = nil
		}
	}
	switch field.Tag {
		// Save camera make and model in case they are needed
		// to identify a maker note.
	case Make:
		rec.make = field.ASCII()
	case Model:
		rec.model = field.ASCII()

		// Old-style JPEG tags have no size fields.
	case JPEGQTables:
		segments := make([]ImageSegment, field.Count)
		for i := uint32(0); i < field.Count; i++ {
			offset := field.Long(i, order)
			size := uint32(64)
			bufsize := uint32(len(buf))
			if offset+size < offset || offset + size > bufsize {
				return nil, &GetIFDError{4, 0, 0, field.Tag}
			}
			segments[i] = buf[offset : offset + size]
		}
		rec.imageData = append(rec.imageData, ImageData{field.Tag, Tag(0), segments})
	case JPEGDCTables, JPEGACTables:
		segments := make([]ImageSegment, field.Count)
		for i := uint32(0); i < field.Count; i++ {
			offset := field.Long(i, order)
			bufsize := uint32(len(buf))
			if offset+16 < offset || offset+16 > bufsize {
				return nil, &GetIFDError{4, 0, 0, field.Tag}
			}
			numvals := uint32(0)
			for j := uint32(0); j < 16; j++ {
				numvals += uint32(buf[offset+j])
			}
			size := 16 + numvals
			segments[i] = buf[offset : offset + size]
		}
		rec.imageData = append(rec.imageData, ImageData{field.Tag, Tag(0), segments})
	}
	return nil, nil
}

func (*TIFFSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (*TIFFSpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (rec TIFFSpaceRec) GetImageData() []ImageData {
	return rec.imageData
}

// Fields in Exif IFDs.
const interOpIFD = 0xA005
const makerNote = 0x927C

// SpaceRec for Exif nodes.
type ExifSpaceRec struct {
	make, model string  // passed from parent TIFF node.
}

func (rec *ExifSpaceRec) GetSpace() TagSpace {
	return ExifSpace
}

func (*ExifSpaceRec) IsMakerNote() bool {
	return false
}

func (*ExifSpaceRec) nodeSize(node IFDNode) uint32 {
	return node.genericSize()
}

// Identify a maker note and return its TagSpace, or TagSpace(0) if not found.
func identifyMakerNote(buf []byte, pos uint32, make, model string) TagSpace {
	var space TagSpace
	lcMake := strings.ToLower(make)
	switch {
	case bytes.HasPrefix(buf[pos:], nikon1Label):
		space = Nikon1Space
	case bytes.HasPrefix(buf[pos:], nikon2LabelPrefix):
		space = Nikon2Space
	case bytes.HasPrefix(buf[pos:], olympus1ALabelPrefix):
		space = Olympus1Space
	case bytes.HasPrefix(buf[pos:], olympus1BLabelPrefix):
		space = Olympus1Space
	case bytes.HasPrefix(buf[pos:], panasonic1Label):
		space = Panasonic1Space

		// If no maker note label was recognized above, assume
		// the maker note is appropriate for the camera make
		// and/or model.
	case strings.HasPrefix(lcMake, "nikon"):
		space = Nikon2Space
	case strings.HasPrefix(lcMake, "canon"):
		space = Canon1Space
	}
	return space
}

func (rec *ExifSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	// SubIFDs.
	if field.Type == IFD || field.Tag == interOpIFD {
		subspace := ExifSpace
		if field.Tag == interOpIFD {
			subspace = InteropSpace
		}
		return recurseSubIFDs(buf, order, ifdPositions, field, NewSpaceRec(subspace))
	}
	// Maker notes
	if field.Tag == makerNote {
		space := identifyMakerNote(buf, dataPos, rec.make, rec.model)
		if space != TagSpace(0) {
			var sub SubIFD
			var err error
			sub.Tag = field.Tag
			sub.Node, err = getIFDTreeIter(buf, order, dataPos, NewSpaceRec(space), ifdPositions)
			if err != nil {
				return nil, err
			}
			return []SubIFD{sub}, nil
		}
	}	
	return nil, nil
}

func (*ExifSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (*ExifSpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (*ExifSpaceRec) GetImageData() []ImageData {
	return nil
}
	
// SpaceRec for Canon1 maker notes.
type Canon1SpaceRec struct {
}

func (*Canon1SpaceRec) GetSpace() TagSpace {
	return Canon1Space
}

func (*Canon1SpaceRec) IsMakerNote() bool {
	return true
}

func (*Canon1SpaceRec) nodeSize(node IFDNode) uint32 {
	return node.genericSize()
}

func (*Canon1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	return nil, nil
}

func (*Canon1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (*Canon1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (*Canon1SpaceRec) GetImageData() []ImageData {
	return nil
}
	
// SpaceRec for Nikon1 maker notes.
type Nikon1SpaceRec struct {
}

func (*Nikon1SpaceRec) GetSpace() TagSpace {
	return Nikon1Space
}

func (*Nikon1SpaceRec) IsMakerNote() bool {
	return true
}

var nikon1Label = []byte("Nikon\000\001\000")

func (*Nikon1SpaceRec) nodeSize(node IFDNode) uint32 {
	return uint32(len(nikon1Label)) + node.genericSize()
}

func (*Nikon1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	return nil, nil
}

func (*Nikon1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos+uint32(len(nikon1Label)), ifdPositions)
}

func (*Nikon1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	copy(buf[pos:], nikon1Label)
	pos += uint32(len(nikon1Label))
	return node.genericPutIFDTree(buf, pos)
}

func (*Nikon1SpaceRec) GetImageData() []ImageData {
	return nil
}
	
// Fields in Nikon2 IFD.
const nikon2PreviewIFD = 0x11
const nikon2NikonScanIFD = 0xE10
const nikon2MakerNoteVersion = 0x1

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

func (*Nikon2SpaceRec) GetSpace() TagSpace {
	return Nikon2Space
}

func (*Nikon2SpaceRec) IsMakerNote() bool {
	return true
}

var nikon2LabelPrefix = []byte("Nikon\000")

func (rec *Nikon2SpaceRec) nodeSize(node IFDNode) uint32 {
	labelLen := len(rec.label)
	if labelLen == 0 {
		// maker note without label or TIFF header.
		return node.genericSize()
	}
	return uint32(labelLen) + HeaderSize + node.genericSize()
}

func (*Nikon2SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	// SubIFDs.
	if field.Type == IFD || field.Tag == nikon2PreviewIFD || field.Tag == nikon2NikonScanIFD {
		subspace := Nikon2Space
		if field.Tag == nikon2PreviewIFD {
			subspace = Nikon2PreviewSpace
		} else if field.Tag == nikon2NikonScanIFD {
			subspace = Nikon2ScanSpace
		}
		return recurseSubIFDs(buf, order, ifdPositions, field, NewSpaceRec(subspace))
	}
	return nil, nil
}

func (rec *Nikon2SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// A few early cameras like Coolpix 775 and 990 use the Nikon
	// 2 tags, but encode the maker note without a label or TIFF
	// header.  If the label is present, the maker note contains a
	// new TIFF block and uses relative offsets within the block.
	if bytes.HasPrefix(buf[pos:], nikon2LabelPrefix) {
		rec.label = buf[pos : int(pos)+len(nikon2LabelPrefix)+4]
		tiff := buf[pos+uint32(len(rec.label)):]
		valid, order, pos := GetHeader(tiff)
		if !valid {
			return errors.New("TIFF header not found in Nikon2 maker note")
		}
		node.Order = order
		return node.genericGetIFDTreeIter(tiff, pos, ifdPositions)		
	} else {
		// Don't assume the endianness is the same as the Exif
		// block. Can work it out by assuming that the number
		// of tags is less than 255.
		if buf[pos] == 0 {
			node.Order = binary.BigEndian
		} else {
			node.Order = binary.LittleEndian
		}
		return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
	}
}

func (rec *Nikon2SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	if len(rec.label) == 0 {
		// maker note without label or TIFF header.
		return node.genericPutIFDTree(buf, pos)
	}
	copy(buf[pos:], rec.label)
	pos += uint32(len(rec.label))
	makerBuf := buf[pos:]
	PutHeader(makerBuf, node.Order, HeaderSize)
	next, err := node.genericPutIFDTree(makerBuf, HeaderSize)
	if err != nil {
		return 0, err
	}
	return pos + next, nil
}

func (rec *Nikon2SpaceRec) GetImageData() []ImageData {
	return nil
}
	
const nikon2PreviewImageStart  = 0x201
const nikon2PreviewImageLength = 0x202

// SpaceRec for Nikon2 Preview IFDs.
type Nikon2PreviewSpaceRec struct {
	offsetField *Field
	lengthField *Field
	imageData []ImageData // May be used for preview image.
}

func (rec *Nikon2PreviewSpaceRec) GetSpace() TagSpace {
	return Nikon2PreviewSpace
}

func (*Nikon2PreviewSpaceRec) IsMakerNote() bool {
	return false
}

func (*Nikon2PreviewSpaceRec) nodeSize(node IFDNode) uint32 {
	return node.genericSize()
}

// Store preview image in the space rec.
func (rec *Nikon2PreviewSpaceRec) appendImageData(buf []byte, order binary.ByteOrder, offsetField, sizeField *Field) error {
	imageData, err := newImageData(buf, order, offsetField, sizeField)
	if err != nil {
		return err
	}
	rec.imageData = append(rec.imageData, *imageData)
	return nil
}

func (rec *Nikon2PreviewSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	// IFD fields aren't usually present in this IFD.
	if field.Type == IFD {
		return recurseSubIFDs(buf, order, ifdPositions, field, NewSpaceRec(Nikon2PreviewSpace))
	}
	if field.Tag == nikon2PreviewImageStart {
		rec.offsetField = field
	} else if field.Tag == nikon2PreviewImageLength {
		rec.lengthField = field
	}
	if rec.offsetField != nil && rec.lengthField != nil {
		rec.appendImageData(buf, order, rec.offsetField, rec.lengthField)
		rec.offsetField = nil
		rec.lengthField = nil
	}
	return nil, nil
}

func (*Nikon2PreviewSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (*Nikon2PreviewSpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (rec *Nikon2PreviewSpaceRec) GetImageData() []ImageData {
	return rec.imageData
}
	
// Fields in Olympus1 IFD.
const olympus1EquipmentIFD = 0x2010
const olympus1CameraSettingsIFD = 0x2020
const olympus1RawDevelopmentIFD = 0x2030
const olympus1RawDev2IFD = 0x2031
const olympus1ImageProcessingIFD = 0x2040
const olympus1FocusInfoIFD = 0x2050

var olympus1ALabelPrefix = []byte("OLYMP\000")  // followed by 2 more bytes
const olympus1ALabelLen uint32 = 8
var olympus1BLabelPrefix = []byte("OLYMPUS\000II") // followed by 2 more bytes
const olympus1BLabelLen uint32 = 12

// SpaceRec for Olympus1 maker notes.
type Olympus1SpaceRec struct {
	// The maker note header/label varies, but the tags are
	// compatible. The older type is decoded with offsets relative
	// to the start of buf and starts with "OLYMP\0", while the
	// newer type is decoded relative to the start of the maker
	// note and starts with ""OLYMPUS\0".  E-M1: "OLYMPUS\0II"
	label []byte
}

func (*Olympus1SpaceRec) GetSpace() TagSpace {
	return Olympus1Space
}

func (*Olympus1SpaceRec) IsMakerNote() bool {
	return true
}

func (rec *Olympus1SpaceRec) nodeSize(node IFDNode) uint32 {
	labelLen := len(rec.label)
	return uint32(labelLen) + node.genericSize()
}

func (*Olympus1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	// SubIFDs.
	if field.Type == IFD || field.Tag == olympus1EquipmentIFD || field.Tag == olympus1CameraSettingsIFD || field.Tag == olympus1RawDevelopmentIFD || field.Tag == olympus1RawDev2IFD || field.Tag == olympus1ImageProcessingIFD || field.Tag == olympus1FocusInfoIFD {
		subspace := Olympus1Space
		switch field.Tag {
		case olympus1EquipmentIFD:
			subspace = Olympus1EquipmentSpace
		case olympus1CameraSettingsIFD:
			subspace = Olympus1CameraSettingsSpace
		case olympus1RawDevelopmentIFD:
			subspace = Olympus1RawDevelopmentSpace
		case olympus1RawDev2IFD:
			subspace = Olympus1RawDev2Space
		case olympus1ImageProcessingIFD:
			subspace = Olympus1ImageProcessingSpace
		case olympus1FocusInfoIFD:
			subspace = Olympus1FocusInfoSpace
		}
		var subIFDs []SubIFD
		var sub SubIFD
		sub.Tag = field.Tag
		var err error
		// In newer maker notes, these fields are IFD type.
		// In older maker notes, they are nominally arrays of
		// UNDEFINED, but contain IFDs that point to data
		// outside the arrays.
		if field.Type == IFD {
			return recurseSubIFDs(buf, order, ifdPositions, field, NewSpaceRec(subspace))
		}
		sub.Node, err = getIFDTreeIter(buf, order, dataPos, NewSpaceRec(subspace), ifdPositions)
		if err != nil {
			return nil, err
		}
		subIFDs = append(subIFDs, sub)
		return subIFDs, nil
	}
	return nil, nil
}

func (rec *Olympus1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	if bytes.HasPrefix(buf[pos:], olympus1ALabelPrefix) {
		rec.label = buf[pos : pos + olympus1ALabelLen]
		// Offsets are relative to start of buf.
		return node.genericGetIFDTreeIter(buf, pos + olympus1ALabelLen, ifdPositions)
	} else if bytes.HasPrefix(buf[pos:], olympus1BLabelPrefix) {
		// Offsets are relative to start of maker note.
		rec.label = buf[pos : pos + olympus1BLabelLen]
		tiff := buf[pos:]
		node.Order = binary.LittleEndian
		return node.genericGetIFDTreeIter(tiff, olympus1BLabelLen, ifdPositions)
	} else {
		return errors.New("Invalid label for Olympus1 maker note")
	}
}

func (rec *Olympus1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	copy(buf[pos:], rec.label)
	if uint32(len(rec.label)) == olympus1ALabelLen {
		pos += uint32(len(rec.label))
		return node.genericPutIFDTree(buf, pos)
	} else if uint32(len(rec.label)) == olympus1BLabelLen {
		makerBuf := buf[pos:]
		next, err := node.genericPutIFDTree(makerBuf, olympus1BLabelLen)
		if err != nil {
			return 0, err
		} else {
			return pos+next, nil
		}
	} else {
		return 0, errors.New("Unexpected Olympus label length")
	}
}

func (*Olympus1SpaceRec) GetImageData() []ImageData {
	return nil
}
	
// SpaceRec for Panasonic1 maker notes.
type Panasonic1SpaceRec struct {
}

func (*Panasonic1SpaceRec) GetSpace() TagSpace {
	return Panasonic1Space
}

func (*Panasonic1SpaceRec) IsMakerNote() bool {
	return true
}

var panasonic1Label = []byte("Panasonic\000\000\000")

func (*Panasonic1SpaceRec) nodeSize(node IFDNode) uint32 {
	return uint32(len(panasonic1Label)) + node.genericSize()
}

func (*Panasonic1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field *Field, dataPos uint32) ([]SubIFD, error) {
	return nil, nil
}

func (*Panasonic1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// Offsets are relative to start of buf.
	return node.genericGetIFDTreeIter(buf, pos+uint32(len(panasonic1Label)), ifdPositions)
}

func (*Panasonic1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	copy(buf[pos:], panasonic1Label)
	pos += uint32(len(panasonic1Label))
	return node.genericPutIFDTree(buf, pos)
}

func (*Panasonic1SpaceRec) GetImageData() []ImageData {
	return nil
}
	
// Put image data from node, if any, into buf at pos. Return next data
// position in buf and a mapping from the offset field tag to an
// encoded array of offsets where image data was placed.
func (node IFDNode) putImageData(buf []byte, order binary.ByteOrder, pos uint32) (uint32, map[Tag][]byte, error) {
	imageData := node.GetImageData()
	if imageData == nil {
		return pos, nil, nil
	}
	offsetTags := make([]Tag, len(imageData))
	for i := range imageData {
		offsetTags[i] = imageData[i].OffsetTag
	}
	offsetFields := node.FindFields(offsetTags)
	if len(offsetFields) != len(offsetTags) {
		return pos, nil, errors.New("putImageData: ImageData offset fields don't match IFD")
	}
	offsetMap := make(map[Tag][]byte)
	for i, id := range imageData {
		if offsetFields[i].Type != LONG && offsetFields[i].Type != SHORT {
			return pos, nil, errors.New("putImageData: OffsetField not LONG or SHORT")
		}
                if id.OffsetTag != offsetFields[i].Tag {
			return pos, nil, errors.New("putImageData: fields not one-to-one")
		}
		offsetData := make([]byte, offsetFields[i].Size())
		offsetMap[offsetTags[i]] = offsetData
		for j, seg := range id.Segments {
			copy(buf[pos:], seg)
			if offsetFields[i].Type == LONG {
				order.PutUint32(offsetData[j*4:], pos)
			} else {
				// Rewriting a file may fail if an offset
				// is in a SHORT field and we are trying to
				// write it too high. The solution is
				// probably to convert such fields to LONG
				// before encoding, IFDNode.Fix().
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

// Specify the serialized position of a SubIFD.
type IFDpos struct {
	Tag  Tag // field that refers to the subIFD.
	Pos  uint32
	Size uint32 // for maker notes only.
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
func (node IFDNode) put(buf []byte, pos uint32, subifds []IFDpos, nextptr uint32) (uint32, error) {
	order := node.Order
	if pos/2*2 != pos {
		return 0, errors.New("IFDNode.Put: pos is not word aligned")
	}
	datapos := pos + node.TableSize()
	// Order in the buffer will be 1) IFD 2) image data 3) IFD external data
	var err error
	datapos, offsets, err := node.putImageData(buf, order, datapos)
	if err != nil {
		return 0, err
	}
	numFields := len(node.Fields)
	order.PutUint16(buf[pos:], uint16(numFields))
	pos += 2
	var lastTag Tag
	var subifdPtrs = make([]*IFDpos, 0, len(subifds))
	for _, field := range node.Fields {
		if field.Tag < lastTag {
			return 0, errors.New(fmt.Sprintf("IFDNode.Put: tags are out of order, %d(0x%X) is followed by %d(0x%X)", lastTag, lastTag, field.Tag, field.Tag))
		}
		lastTag = field.Tag
		order.PutUint16(buf[pos:], uint16(field.Tag))
		pos += 2
		order.PutUint16(buf[pos:], uint16(field.Type))
		pos += 2
		// We can handle two kinds of subIFDs. Firstly, fields
		// that just contain a pointer to one or more subIFDs,
		// usually in a field with LONG or IFD type. Secondly,
		// IFDs stored in arrays of a field which has a type
		// size of 1, typically UNDEFINED, e.g., maker notes.
		subifdPtrs = subifdPtrs[0:0]
		for i := range subifds {
			if subifds[i].Tag == field.Tag {
				subifdPtrs = append(subifdPtrs, &subifds[i])
			}
		}
		if len(subifdPtrs) > 0 && field.Type.Size() == 1 {
			if len(subifdPtrs) > 1 {
				errors.New("IFDNode.Put: IFD array field expected to have a single IFD.")
			}
			if subifdPtrs[0].Size < 5 {
				errors.New("IFDNode.Put: sub-IFD expected to have size > 4")
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
				return 0, errors.New("IFDNode.Put: sub-IFD pointer expected to have field type with size 4")
			}
			data = make([]byte, size)
			for i := range subifdPtrs {
				order.PutUint32(data[i*4:], subifdPtrs[i].Pos)
			}
		} else {
			fieldOffsets := offsets[field.Tag]
			if fieldOffsets != nil {
				// Image data offset field.
				data = fieldOffsets
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

// Serialize an IFD and all the other IFDs to which it refers into a
// byte slice at 'pos'.  Returns the position following the last byte
// used. 'buf' must represent a serialized file with the start of the
// file at the start of the slice, and it must be sufficiently large
// for the new data. 'pos' must be word (2 byte) aligned and the tags
// in the IFDs must be in ascending order, according to the TIFF
// specification.
func (node IFDNode) PutIFDTree(buf []byte, pos uint32) (uint32, error) {
	// Allow the PutIFDTree function to be selected according to
	// the node space. Normal TIFF nodes will call
	// genericPutIFDTree below.
	return node.SpaceRec.putIFDTree(node, buf, pos)
}

// Version of PutIFDTree without special processing for things like
// maker note labels.
func (node IFDNode) genericPutIFDTree(buf []byte, pos uint32) (uint32, error) {
	// Write node's IFD at pos. But first write any IFDs that it
	// refers to, recording their positions.
	nsubs := len(node.SubIFDs)
	subpos := make([]IFDpos, nsubs)
	next := pos + node.genericSize()
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
	_, err = node.put(buf, pos, subpos, nextPos)
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
func (node *IFDNode) fixIFD() {
	sort.Slice(node.Fields, func(i, j int) bool { return node.Fields[i].Tag < node.Fields[j].Tag })
	imageData := node.GetImageData()
	for _, field := range node.Fields {
		if field.Type == SHORT {
			for j := range imageData {
				if imageData[j].OffsetTag == field.Tag {
					offsets := make([]uint32, field.Count)
					for k := uint32(0); k < field.Count; k++ {
						offsets[k] = uint32(field.Short(k, node.Order))
					}
					field.Type = LONG
					field.Data = make([]byte, 4*field.Count)
					for k := uint32(0); k < field.Count; k++ {
						field.PutLong(offsets[k], k, node.Order)
					}
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
	node.fixIFD()
	for i := 0; i < len(node.SubIFDs); i++ {
		node.SubIFDs[i].Node.Fix()
	}
	if node.Next != nil {
		node.Next.Fix()
	}
}
