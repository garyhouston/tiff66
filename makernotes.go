package tiff66

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strings"
)

// Identify a maker note and return its TagSpace, or TagSpace(0) if not found.
func identifyMakerNote(buf []byte, pos uint32, make, model string) TagSpace {
	var space TagSpace
	lcMake := strings.ToLower(make)
	switch {
	case bytes.HasPrefix(buf[pos:], fujifilm1Label):
		space = Fujifilm1Space
	case bytes.HasPrefix(buf[pos:], generaleLabel):
		space = Fujifilm1Space
	case bytes.HasPrefix(buf[pos:], nikon1Label):
		space = Nikon1Space
	case bytes.HasPrefix(buf[pos:], nikon2LabelPrefix):
		space = Nikon2Space
	case bytes.HasPrefix(buf[pos:], panasonic1Label):
		space = Panasonic1Space
	default:
		for i := range olympus1Labels {
			if bytes.HasPrefix(buf[pos:], olympus1Labels[i].prefix) {
				space = Olympus1Space
			}
		}
		if space == TagSpace(0) {
			for i := range sony1Labels {
				if bytes.HasPrefix(buf[pos:], sony1Labels[i]) {
					space = Sony1Space
				}
			}
		}
		// If no maker note label was recognized above, assume
		// the maker note is appropriate for the camera make
		// and/or model.
		if space == TagSpace(0) {
			switch {
			case strings.HasPrefix(lcMake, "nikon"):
				space = Nikon2Space
			case strings.HasPrefix(lcMake, "canon"):
				space = Canon1Space
			}
		}
	}
	return space
}

// Given a buffer pointing to a an IFD entry count, guess the byte
// order of the IFD. The number of entries is usually small,
// usually less than 256.
func detectByteOrder(buf []byte) binary.ByteOrder {
	big := binary.BigEndian.Uint16(buf)
	little := binary.LittleEndian.Uint16(buf)
	if little < big {
		return binary.LittleEndian
	} else {
		return binary.BigEndian
	}
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

func (*Canon1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	return nil, nil
}

func (*Canon1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (*Canon1SpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.unexpectedFooter(buf, pos, ifdPositions)
}

func (*Canon1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	return node.genericPutIFDTree(buf, pos)
}

func (*Canon1SpaceRec) GetImageData() []ImageData {
	return nil
}

// SpaceRec for Fujifilm1 maker notes.
type Fujifilm1SpaceRec struct {
	label []byte
}

func (*Fujifilm1SpaceRec) GetSpace() TagSpace {
	return Fujifilm1Space
}

func (*Fujifilm1SpaceRec) IsMakerNote() bool {
	return true
}

var fujifilm1Label = []byte("FUJIFILM")
var generaleLabel = []byte("GENERALE") // GE E1255W

func (rec *Fujifilm1SpaceRec) nodeSize(node IFDNode) uint32 {
	// Label, IFD position, and IFD.
	return uint32(len(rec.label)) + 4 + node.genericSize()
}

func (*Fujifilm1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	return nil, nil
}

func (rec *Fujifilm1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// Offsets are relative to start of the makernote.
	tiff := buf[pos:]
	if bytes.HasPrefix(tiff, fujifilm1Label) {
		rec.label = append([]byte{}, fujifilm1Label...)
	} else if bytes.HasPrefix(tiff, generaleLabel) {
		rec.label = append([]byte{}, generaleLabel...)
	} else {
		// Shouldn't reach this point if we already know it's a Fujifilm1SpaceRec.
		return errors.New("Invalid label for Fujifilm1 maker note")
	}
	// Must be read as little-endian, even if the Exif block is
	// big endian (as in Leica digilux 4.3).
	node.Order = binary.LittleEndian
	// Only the 2nd half of the TIFF header is present, the position
	// of the IFD.
	pos = node.Order.Uint32(tiff[len(rec.label):])
	return node.genericGetIFDTreeIter(tiff, pos, ifdPositions)
}

func (*Fujifilm1SpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.unexpectedFooter(buf, pos, ifdPositions)
}

func (rec *Fujifilm1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	tiff := buf[pos:]
	copy(tiff, rec.label)
	lablen := uint32(len(rec.label))
	start := lablen + 4
	node.Order.PutUint32(tiff[lablen:], start)
	next, err := node.genericPutIFDTree(tiff, start)
	if err != nil {
		return 0, err
	}
	return pos + next, nil
}

func (*Fujifilm1SpaceRec) GetImageData() []ImageData {
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

func (*Nikon1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	return nil, nil
}

func (*Nikon1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos+uint32(len(nikon1Label)), ifdPositions)
}

func (*Nikon1SpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.unexpectedFooter(buf, pos, ifdPositions)
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
	// Nikon D5100: "Nikon\0\2\x10\0\0"
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

func (*Nikon2SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
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
		lablen := uint32(len(nikon2LabelPrefix) + 4)
		rec.label = append([]byte{}, buf[pos:pos+lablen]...)
		tiff := buf[pos+lablen:]
		valid, order, pos := GetHeader(tiff)
		if !valid {
			return errors.New("TIFF header not found in Nikon2 maker note")
		}
		node.Order = order
		return node.genericGetIFDTreeIter(tiff, pos, ifdPositions)
	} else {
		// Byte order may differ from Exif block.
		node.Order = detectByteOrder(buf[pos:])
		return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
	}
}

func (*Nikon2SpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.unexpectedFooter(buf, pos, ifdPositions)
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

const nikon2PreviewImageStart = 0x201
const nikon2PreviewImageLength = 0x202

// SpaceRec for Nikon2 Preview IFDs.
type Nikon2PreviewSpaceRec struct {
	offsetField Field
	lengthField Field
	imageData   []ImageData // May be used for preview image.
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
func (rec *Nikon2PreviewSpaceRec) appendImageData(buf []byte, order binary.ByteOrder, offsetField, sizeField Field) error {
	imageData, err := newImageData(buf, order, offsetField, sizeField)
	if err != nil {
		return err
	}
	rec.imageData = append(rec.imageData, *imageData)
	return nil
}

func (rec *Nikon2PreviewSpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	// IFD fields aren't usually present in this IFD.
	if field.Type == IFD {
		return recurseSubIFDs(buf, order, ifdPositions, field, NewSpaceRec(Nikon2PreviewSpace))
	}
	if field.Tag == nikon2PreviewImageStart {
		rec.offsetField = field
	} else if field.Tag == nikon2PreviewImageLength {
		rec.lengthField = field
	}
	if rec.offsetField.Tag != 0 && rec.lengthField.Tag != 0 {
		rec.appendImageData(buf, order, rec.offsetField, rec.lengthField)
		rec.offsetField.Tag = 0
		rec.lengthField.Tag = 0
	}
	return nil, nil
}

func (*Nikon2PreviewSpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.genericGetIFDTreeIter(buf, pos, ifdPositions)
}

func (*Nikon2PreviewSpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.unexpectedFooter(buf, pos, ifdPositions)
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
const olympus1FocusInfo = 0x2050

// The Olympus1 maker note header/label varies, but the tags are
// compatible. The older type is decoded with offsets relative to the
// start of the Tiff block and starts with "OLYMP\0" or other labels;
// the newer type is decoded relative to the start of the maker note
// and starts with ""OLYMPUS\000II".

var olympus1Labels = []struct {
	prefix   []byte // Identifying prefix of maker note label.
	length   uint32 // Full length of maker note label.
	relative bool   // True if offsets are relative to the start of the maker note, instead of the entire Tiff block.
}{
	{[]byte("OLYMP\000"), 8, false},     // Many Olympus models.
	{[]byte("OLYMPUS\000II"), 12, true}, // Many Olympus models.
	{[]byte("SONY PI\000"), 12, false},  // Sony DSC-S650 etc.
	{[]byte("PREMI\000"), 8, false},     // Sony DSC-S45, DSC-S500.
	{[]byte("CAMER\000"), 8, false},     // Various Premier models, sometimes rebranded.
	{[]byte("MINOL\000"), 8, false},     // Minolta DiMAGE E323.
}

// SpaceRec for Olympus1 maker notes.
type Olympus1SpaceRec struct {
	label    []byte
	relative bool // True if offsets relative to start of maker note, instead of entire Tiff block.
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

func (*Olympus1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	// SubIFDs.
	if field.Type == IFD || field.Tag == olympus1EquipmentIFD || field.Tag == olympus1CameraSettingsIFD || field.Tag == olympus1RawDevelopmentIFD || field.Tag == olympus1RawDev2IFD || field.Tag == olympus1ImageProcessingIFD || field.Tag == olympus1FocusInfo {
		if field.Tag == olympus1FocusInfo && field.Type == UNDEFINED {
			// Some camera models make this is an IFD, but in others it's just an array of data. Make a guess.
			// The field size is often just the tableEntrySize times the number of entries
			// in the table. I.e., it omits the table overhead and the external data.
			if field.Size() < tableEntrySize {
				// Too small to be an IFD.
				return nil, nil
			}
			data := field.Data
			entries := order.Uint16(data)
			if entries == 0 {
				// IFD should have entries.
				return nil, nil
			}
			if field.Size() < uint32(entries)*tableEntrySize {
				// Field is too small to be an IFD with the specified number of fields.
				return nil, nil
			}
			end := dataPos + tableSize(entries)
			if end < dataPos || end > uint32(len(buf)) {
				// IFD with specified number of fields would run past end of buffer.
				return nil, nil
			}
			check := uint16(3) // Check the types of the first fields, allow for slightly damaged IFDs.
			if check > entries {
				check = entries
			}
			for i := uint16(0); i < check; i++ {
				typ := Type(order.Uint16(data[2+i*tableEntrySize+2:]))
				if typ == 0 || typ > IFD {
					// Not an offficial data type: probably not an IFD field.
					return nil, nil
				}
			}
		}
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
		case olympus1FocusInfo:
			subspace = Olympus1FocusInfoSpace
		}
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
		return []SubIFD{sub}, err
	}
	return nil, nil
}

func (rec *Olympus1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	for i := range olympus1Labels {
		if bytes.HasPrefix(buf[pos:], olympus1Labels[i].prefix) {
			rec.label = append([]byte{}, buf[pos:pos+olympus1Labels[i].length]...)
			// Byte order varies by camera model, and may differ from Exif order.
			node.Order = detectByteOrder(buf[pos+olympus1Labels[i].length:])
			if olympus1Labels[i].relative {
				// Offsets are relative to start of maker note.
				tiff := buf[pos:]
				rec.relative = true
				return node.genericGetIFDTreeIter(tiff, olympus1Labels[i].length, ifdPositions)
			} else {
				// Offsets are relative to start of buffer.
				rec.relative = false
				return node.genericGetIFDTreeIter(buf, pos+olympus1Labels[i].length, ifdPositions)
			}
		}
	}
	// Shouldn't reach this point if we already know it's an Olympus1SpaceRec.
	return errors.New("Invalid label for Olympus1 maker note")
}

func (*Olympus1SpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	return node.unexpectedFooter(buf, pos, ifdPositions)
}

func (rec *Olympus1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	copy(buf[pos:], rec.label)
	labelLen := uint32(len(rec.label))
	if rec.relative {
		makerBuf := buf[pos:]
		next, err := node.genericPutIFDTree(makerBuf, labelLen)
		if err != nil {
			return 0, err
		} else {
			return pos + next, nil
		}
	} else {
		pos += uint32(labelLen)
		return node.genericPutIFDTree(buf, pos)
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

func (*Panasonic1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	return nil, nil
}

func (*Panasonic1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// Offsets are relative to start of buf.
	return node.genericGetIFDTreeIter(buf, pos+uint32(len(panasonic1Label)), ifdPositions)
}

func (rec *Panasonic1SpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// Next pointer is generally missing, don't try to read it.
	return nil
}

func (*Panasonic1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	copy(buf[pos:], panasonic1Label)
	pos += uint32(len(panasonic1Label))
	return node.genericPutIFDTree(buf, pos)
}

func (*Panasonic1SpaceRec) GetImageData() []ImageData {
	return nil
}

// SpaceRec for Sony1 maker notes.
type Sony1SpaceRec struct {
	label []byte
}

func (*Sony1SpaceRec) GetSpace() TagSpace {
	return Sony1Space
}

func (*Sony1SpaceRec) IsMakerNote() bool {
	return true
}

var sony1Labels = [][]byte{
	[]byte("SONY CAM \000\000\000"),    // Includes various Sony camcorders.
	[]byte("SONY DSC \000\000\000"),    // Includes various Sony still cameras.
	[]byte("\000\000SONY PIC\000\000"), // Sony DSC-TF1.
	[]byte("SONY MOBILE\000"),          // Sony Xperia.
	[]byte("VHAB     \000\000\000"),    // Hasselblad versions of Sony cameras.
}

func (rec *Sony1SpaceRec) nodeSize(node IFDNode) uint32 {
	return uint32(len(rec.label)) + node.genericSize()
}

func (*Sony1SpaceRec) takeField(buf []byte, order binary.ByteOrder, ifdPositions posMap, idx uint16, field Field, dataPos uint32) ([]SubIFD, error) {
	return nil, nil
}

func (rec *Sony1SpaceRec) getIFDTree(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	for _, label := range sony1Labels {
		if bytes.HasPrefix(buf[pos:], label) {
			rec.label = append([]byte{}, label...)
			ifdpos := pos + uint32(len(rec.label))
			// Byte order varies by camera model, and may differ from Exif order.
			node.Order = detectByteOrder(buf[ifdpos:])
			return node.genericGetIFDTreeIter(buf, ifdpos, ifdPositions)
		}
	}
	// Shouldn't reach this point if we already know it's a Sony1SpaceRec.
	return errors.New("Invalid label for Sony1 maker note")
}

func (rec *Sony1SpaceRec) getFooter(node *IFDNode, buf []byte, pos uint32, ifdPositions posMap) error {
	// Next pointer is often invalid, don't try to read it.
	return nil
}

func (rec *Sony1SpaceRec) putIFDTree(node IFDNode, buf []byte, pos uint32) (uint32, error) {
	copy(buf[pos:], rec.label)
	pos += uint32(len(rec.label))
	return node.genericPutIFDTree(buf, pos)
}

func (*Sony1SpaceRec) GetImageData() []ImageData {
	return nil
}
