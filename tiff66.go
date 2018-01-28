package tiff66

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"math"
	"sort"
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

// Node in an IFD tree.
type IFDNode struct {
	// Usually all IFDs in a TIFF file have the same byte order,
	// specified at the start of the file, but this may not be
	// the case for maker notes.
	Order  binary.ByteOrder
	Fields []Field
	SpaceRec
	SubIFDs []SubIFD // Links to sub-IFD nodes linked by fields.
	Next    *IFDNode // Tail link to next node.
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

const tableOverhead = 6 // 2 bytes for the entry count and 4 for the position of the next IFD.
const tableEntrySize = 12

// Return the serialized size of a basic IFD table.
func tableSize(numFields uint16) uint32 {
	return tableOverhead + uint32(numFields)*tableEntrySize
}

func (node IFDNode) TableSize() uint32 {
	return tableSize(uint16(len(node.Fields)))
}

// Return the number of IFD table entries that would fit in size bytes.
func maxTableEntries(size uint32) uint32 {
	return (size - tableOverhead) / tableEntrySize
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
// TIFFSpace. It will try to read as much data as possible, even if there
// are errors. If no useful data can be obtained, the returned node will
// have Fields with len 0, possibly nil, and possibly with a pointer to
// the next IFD. The error may be a multierror structure.
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

// Version of getIFDTreeIter without subspace-specific header processing. Try to read fields and process sub-IFDs.
func (node *IFDNode) genericGetIFDTreeIter(buf []byte, pos uint32, ifdPositions posMap) error {
	space := node.GetSpace()
	// ifdpos is the byte position in the file, except in certain maker notes.
	ifdpos := pos
	if ifdPositions[posKey(buf, pos)] {
		return fmt.Errorf("IFD cycle detected in %s IFD at %d", space.Name(), ifdpos)
	}
	ifdPositions[posKey(buf, pos)] = true
	node.SubIFDs = make([]SubIFD, 0, 10)
	bufsize := uint32(len(buf))
	if pos+2 < pos || pos+2 > bufsize {
		return fmt.Errorf("Could not read %s IFD at %d: past end of input", space.Name(), ifdpos)
	}
	order := node.Order
	// Whether to process the pointer at the end of the IFD that points to the next one.
	processNext := true
	entries := order.Uint16(buf[pos:]) // IFD entry count.
	var err error
	if entries == 0 {
		// Technically an error since the TIFF spec doesn't permit IFDs with no entries. There may still be
		// a Next pointer.
		err = multierror.Append(err, fmt.Errorf("%s IFD at %d doesn't contain any fields", space.Name(), ifdpos))
	}
	tabsize := tableSize(entries)
	if pos+tabsize < pos || pos+tabsize > bufsize {
		processNext = false
		// The table extends past the end of the buffer:
		// examine the table for possibly valid fields - tags
		// should be increasing in value.
		entries = uint16(maxTableEntries(bufsize - pos))
		for i, last := uint16(0), Tag(0); i < entries; i++ {
			tagpos := pos + 2 + uint32(i*tableEntrySize)
			tag := Tag(order.Uint16(buf[tagpos:]))
			if tag < last {
				entries = i
				break
			}
			last = tag
		}
		err = multierror.Append(err, fmt.Errorf("%s IFD at %d extends past end of input, attempting to read %d entries", space.Name(), ifdpos, entries))
	}
	pos += 2
	fields := make([]Field, 0, entries)
	for i := uint16(0); i < entries; i++ {
		var field Field
		field.Tag = Tag(order.Uint16(buf[pos:]))
		pos += 2
		field.Type = Type(order.Uint16(buf[pos:]))
		pos += 2
		field.Count = order.Uint32(buf[pos:])
		pos += 4
		size := field.Size()
		dataPos := pos
		pos += 4
		if size > 4 {
			dataPos = order.Uint32(buf[dataPos:])
			if dataPos+size < dataPos || dataPos+size > bufsize {
				err = multierror.Append(err, fmt.Errorf("Skipping field %d with tag %d (0x%0X) in %s IFD at %d: data at %d past end of input", i, field.Tag, field.Tag, space.Name(), ifdpos, dataPos))
				continue
			}
		}
		field.Data = buf[dataPos : dataPos+size]
		// Space-specific field processing, including subIFD recursion.
		subIFDs, fieldErr := node.SpaceRec.takeField(buf, order, ifdPositions, i, field, dataPos)
		if fieldErr != nil {
			err = multierror.Append(err, fieldErr)
		}
		if subIFDs != nil {
			node.SubIFDs = append(node.SubIFDs, subIFDs...)
		}
		fields = append(fields, field)
	}
	node.Fields = fields
	if processNext {
		footerErr := node.SpaceRec.getFooter(node, buf, pos, ifdPositions)
		if footerErr != nil {
			err = multierror.Append(err, footerErr)
		}
	}
	return err
}

// Generic processing of the "next" pointer at the end of an IFD. Modifies node.
func (node *IFDNode) genericGetFooter(buf []byte, pos uint32, nextSpace TagSpace, ifdPositions posMap) error {
	buflen := uint32(len(buf))
	space := node.GetSpace()
	if pos+4 < pos || pos+4 > buflen {
		// This shouldn't happen, since table size is checked earlier.
		return fmt.Errorf("Can't read Next pointer in %s IFD; past end of input", space.Name())
	}
	next := node.Order.Uint32(buf[pos:])
	if next > 0 {
		if next >= buflen {
			return fmt.Errorf("Next pointer %d in %s IFD past end of input", next, space.Name())
		}
		var err error
		node.Next, err = getIFDTreeIter(buf, node.Order, next, NewSpaceRec(nextSpace), ifdPositions)
		return err
	}
	return nil
}

// Similar to genericGetFooter, but additionally add an error if a next IFD is found.
func (node *IFDNode) unexpectedFooter(buf []byte, pos uint32, ifdPositions posMap) error {
	buflen := uint32(len(buf))
	space := node.GetSpace()
	if pos+4 < pos || pos+4 > buflen {
		// This shouldn't happen, since table size is checked earlier.
		return fmt.Errorf("Can't read Next pointer in %s IFD; past end of input", space.Name())
	}
	next := node.Order.Uint32(buf[pos:])
	if next != 0 {
		err := fmt.Errorf("Unexpected pointer %d to next IFD in %s IFD", next, space.Name())
		// Unexpected, but process it anyway.
		return multierror.Append(err, node.genericGetFooter(buf, pos, space, ifdPositions))
	}
	return nil
}

// IFD Tag namespace.
type TagSpace uint8

const (
	TIFFSpace                    TagSpace = 0
	UnknownSpace                 TagSpace = 1
	ExifSpace                    TagSpace = 2
	GPSSpace                     TagSpace = 3
	InteropSpace                 TagSpace = 4
	MPFIndexSpace                TagSpace = 5 // Multi-Picture Format.
	MPFAttributeSpace            TagSpace = 6
	Canon1Space                  TagSpace = 7
	Fujifilm1Space               TagSpace = 20
	Nikon1Space                  TagSpace = 8
	Nikon2Space                  TagSpace = 9
	Nikon2PreviewSpace           TagSpace = 10
	Nikon2ScanSpace              TagSpace = 11
	Olympus1Space                TagSpace = 12
	Olympus1EquipmentSpace       TagSpace = 13
	Olympus1CameraSettingsSpace  TagSpace = 14
	Olympus1RawDevelopmentSpace  TagSpace = 15
	Olympus1RawDev2Space         TagSpace = 16
	Olympus1ImageProcessingSpace TagSpace = 17
	Olympus1FocusInfoSpace       TagSpace = 18
	Panasonic1Space              TagSpace = 19
	Sony1Space                   TagSpace = 21 // last
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
	case Fujifilm1Space:
		return "Fujifilm1"
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
	case Sony1Space:
		return "Sony1"
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

// An interface for node-space-specific functionality.
type SpaceRec interface {
	GetSpace() TagSpace
	IsMakerNote() bool
	nodeSize(IFDNode) uint32
	takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error)
	getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error
	// Called by getIFDTree to process the part of the IFD
	// following the field entries, usually 4 bytes with the next
	// IFD or zero. The next IFD will be read recursively.
	getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error
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
	case Fujifilm1Space:
		return &Fujifilm1SpaceRec{}
	case MPFIndexSpace:
		return &MPFIndexSpaceRec{}
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
	case Sony1Space:
		return &Sony1SpaceRec{}
	default:
		// Don't expect Next pointers to be present in any of the
		// known IFDs, but permit them in unknown IFDs.
		if space != UnknownSpace {
			return &NoNextSpaceRec{space: space}
		}
		return &GenericSpaceRec{space: space}
	}
}

// Recursively read SubIFDs specified with a given field. Such fields
// contain pointer(s) to the SubIFD location(s).
func recurseSubIFDs(buf []byte, order binary.ByteOrder, ifdPositions posMap, field Field, spaceRec SpaceRec) ([]SubIFD, error) {
	var subIFDs []SubIFD
	var err error
	for i := uint32(0); i < field.Count; i++ {
		var sub SubIFD
		sub.Tag = field.Tag
		var suberr error
		sub.Node, suberr = getIFDTreeIter(buf, order, field.Long(i, order), spaceRec, ifdPositions)
		if suberr != nil {
			err = multierror.Append(err, suberr)
		}
		subIFDs = append(subIFDs, sub)
	}
	return subIFDs, err
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

func (rec *GenericSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
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

func (rec *GenericSpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// Assume any following IFD has the same space as the current.
	return node.genericGetFooter(buf, pos, rec.space, ifdPositions)
}

func (*GenericSpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (*GenericSpaceRec) GetImageData() []ImageData {
	return nil
}

// Similar to GenericSpaceRec, but don't process Next pointers: not
// expected to be present for these spaces.
type NoNextSpaceRec struct {
	space TagSpace
}

func (rec *NoNextSpaceRec) GetSpace() TagSpace {
	return rec.space
}

func (*NoNextSpaceRec) IsMakerNote() bool {
	return false
}

func (*NoNextSpaceRec) nodeSize(node IFDNode) uint32 {
	return node.genericSize()
}

func (rec *NoNextSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	// Process a field of type IFD: these declare a subIFD, and
	// can be potentially found in any IFD.  Assume the subIFD has
	// the same space as the current IFD.
	if field.Type == IFD {
		return recurseSubIFDs(buf, order, ifdPositions, field, NewSpaceRec(rec.space))
	}
	return nil, nil
}

func (*NoNextSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (rec *NoNextSpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.unexpectedFooter(buf, pos, ifdPositions)
}

func (*NoNextSpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (*NoNextSpaceRec) GetImageData() []ImageData {
	return nil
}

var tiffOffsetTags = []Tag{StripOffsets, TileOffsets, FreeOffsets, JPEGInterchangeFormat}
var tiffSizeTags = []Tag{StripByteCounts, TileByteCounts, FreeByteCounts, JPEGInterchangeFormatLength}

const tiffNumTags = 4

// SpaceRec for TIFF nodes.
type TIFFSpaceRec struct {
	offsetFields [tiffNumTags]Field
	sizeFields   [tiffNumTags]Field
	make, model  string
	imageData    []ImageData
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

func newImageData(buf []byte, order binary.ByteOrder, offsetField, sizeField Field) (*ImageData, error) {
	segments := make([]ImageSegment, offsetField.Count)
	for i := uint32(0); i < offsetField.Count; i++ {
		offset := uint32(offsetField.AnyInteger(i, order))
		size := uint32(sizeField.AnyInteger(i, order))
		bufsize := uint32(len(buf))
		if offset+size < offset || offset+size > bufsize {
			return nil, fmt.Errorf("Image data for tags %d / %d extends past end of input", offsetField.Tag, sizeField.Tag)
		}
		segments[i] = buf[offset : offset+size]
	}
	return &ImageData{offsetField.Tag, sizeField.Tag, segments}, nil
}

// Store image data in the TIFF space rec.
func (rec *TIFFSpaceRec) appendImageData(buf []byte, order binary.ByteOrder, offsetField, sizeField Field) error {
	imageData, err := newImageData(buf, order, offsetField, sizeField)
	if err != nil {
		return err
	}
	rec.imageData = append(rec.imageData, *imageData)
	return nil
}

func (rec *TIFFSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	// SubIFDs.
	if field.Type == IFD || field.Tag == SubIFDs || field.Tag == ExifIFD || field.Tag == GPSIFD {
		var spaceRec SpaceRec
		if field.Tag == ExifIFD {
			spaceRec = &ExifSpaceRec{make: rec.make, model: rec.model}
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
		if rec.offsetFields[i].Tag != 0 && rec.sizeFields[i].Tag != 0 {
			rec.appendImageData(buf, order, rec.offsetFields[i], rec.sizeFields[i])
			rec.offsetFields[i].Tag = 0
			rec.sizeFields[i].Tag = 0
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
			if offset+size < offset || offset+size > bufsize {
				return nil, fmt.Errorf("Image data for tag %d extends past end of input", field.Tag)
			}
			segments[i] = buf[offset : offset+size]
		}
		rec.imageData = append(rec.imageData, ImageData{field.Tag, Tag(0), segments})
	case JPEGDCTables, JPEGACTables:
		segments := make([]ImageSegment, field.Count)
		for i := uint32(0); i < field.Count; i++ {
			offset := field.Long(i, order)
			bufsize := uint32(len(buf))
			if offset+16 < offset || offset+16 > bufsize {
				return nil, fmt.Errorf("Image data for tag %d extends past end of input", field.Tag)
			}
			numvals := uint32(0)
			for j := uint32(0); j < 16; j++ {
				numvals += uint32(buf[offset+j])
			}
			size := 16 + numvals
			segments[i] = buf[offset : offset+size]
		}
		rec.imageData = append(rec.imageData, ImageData{field.Tag, Tag(0), segments})
	}
	return nil, nil
}

func (*TIFFSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (*TIFFSpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetFooter(buf, pos, node.GetSpace(), ifdPositions)
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
	make, model string // passed from parent TIFF node.
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

func (rec *ExifSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
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
			return []SubIFD{sub}, err
		}
	}
	return nil, nil
}

func (*ExifSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (rec *ExifSpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// The next IFD after an Exif IFD is a thumbnail encoded as
	// TIFF.
	return node.genericGetFooter(buf, pos, TIFFSpace, ifdPositions)
}

func (*ExifSpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (*ExifSpaceRec) GetImageData() []ImageData {
	return nil
}

// SpaceRec for MPFIndex nodes. Similar to Generic but with specific Next processing.
type MPFIndexSpaceRec struct {
	space TagSpace
}

func (rec *MPFIndexSpaceRec) GetSpace() TagSpace {
	return rec.space
}

func (*MPFIndexSpaceRec) IsMakerNote() bool {
	return false
}

func (*MPFIndexSpaceRec) nodeSize(node IFDNode) uint32 {
	return node.genericSize()
}

func (rec *MPFIndexSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	// Process a field of type IFD: these declare a subIFD, and
	// can be potentially found in any IFD.  Assume the subIFD has
	// the same space as the current IFD.
	if field.Type == IFD {
		return recurseSubIFDs(buf, order, ifdPositions, field, NewSpaceRec(rec.space))
	}
	return nil, nil
}

func (*MPFIndexSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (rec *MPFIndexSpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// MPFIndex space may be followd by an MPFAttribute space.
	return node.genericGetFooter(buf, pos, MPFAttributeSpace, ifdPositions)
}

func (*MPFIndexSpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (*MPFIndexSpaceRec) GetImageData() []ImageData {
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
			return 0, fmt.Errorf("IFDNode.Put: tags are out of order, %d(0x%X) is followed by %d(0x%X)", lastTag, lastTag, field.Tag, field.Tag)
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
				return 0, errors.New("IFDNode.Put: IFD array field expected to have a single IFD.")
			}
			if subifdPtrs[0].Size < 5 {
				return 0, errors.New("IFDNode.Put: sub-IFD expected to have size > 4")
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
			if len(subifdPtrs) != int(field.Count) {
				return 0, fmt.Errorf("IFDNode.Put: field (%d) count (%d) doesn't match number of sub-IFDs (%d)", field.Tag, field.Count, len(subifdPtrs))
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
			if field.Count > 0 && field.Data[field.Count-1] != 0 {
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

// Delete the nth SubIFD from a node, also removing its reference in the fields.
func (node *IFDNode) DeleteSubIFD(n int) {
	for i := range node.Fields {
		if node.Fields[i].Tag == node.SubIFDs[n].Tag {
			if node.Fields[i].Type.Size() == 1 {
				// Fields of byte type where the Count is the packed size of a single subIFD.
				node.DeleteFields([]Tag{node.Fields[i].Tag})
			} else {
				// Fields of integer type where the Count is the number of subIFDs.
				node.Fields[i].Count--
				if node.Fields[i].Count == 0 {
					node.DeleteFields([]Tag{node.Fields[i].Tag})
				}
			}
			break
		}
	}
	node.SubIFDs = append(node.SubIFDs[:n], node.SubIFDs[n+1:]...)
}

// Remove nodes with no fields, which are prohibited by the TIFF spec (1992),
// Secton 2: TIFF Structure. Image File Directory. "There must be at least 1 IFD
// in a TIFF file and each IFD must have at least one entry." Returns the modified
// node, or nil if it contains no fields.
func (node *IFDNode) DeleteEmptyIFDs() *IFDNode {
	for i := 0; i < len(node.SubIFDs); i++ {
		node.SubIFDs[i].Node = node.SubIFDs[i].Node.DeleteEmptyIFDs()
		if node.SubIFDs[i].Node == nil {
			node.DeleteSubIFD(i)
			i-- // Process this index again, it will now refer to the next subIFD.
		}
	}
	if len(node.Fields) == 0 {
		if node.Next == nil {
			return nil
		}
		return node.Next.DeleteEmptyIFDs()
	}
	if node.Next != nil {
		node.Next = node.Next.DeleteEmptyIFDs()
	}
	return node
}
