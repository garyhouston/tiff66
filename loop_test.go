package tiff66

import (
	"encoding/binary"
	"strings"
	"testing"
)

// Create a TIFF buffer where one IFD refers to another and vice versa, and
// check that reading it back gives an error.
func TestLoop(t *testing.T) {
	order := binary.LittleEndian
	var ifd1 IFD_T
	ifd1.Fields = make([]Field, 1)
	ifd1.Fields[0] = Field{Compression, BYTE, 1, nil}
	ifd1.Fields[0].Data = []byte("\001")
	var ifd2 IFD_T
	ifd2.Fields = make([]Field, 1)
	ifd2.Fields[0] = ifd1.Fields[0]
	ifdsize := ifd1.Size() // ifd contains no external data
	buf := make([]byte, HeaderSize+2*ifdsize)
	ifd1pos := uint32(HeaderSize)
	ifd2pos := ifd1pos + ifdsize
	PutHeader(buf, order, ifd1pos)
	_, err := ifd1.Put(buf, order, ifd1pos, nil, ifd2pos)
	if err != nil {
		t.Error("Failed to put ifd1")
	}
	_, err = ifd2.Put(buf, order, ifd2pos, nil, ifd1pos)
	if err != nil {
		t.Error("Failed to put ifd2")
	}
	valid, getorder, getpos := GetHeader(buf)
	if !valid {
		t.Error("Header not valid")
	}
	_, err = GetIFDTree(buf, getorder, getpos, TIFFSpace)
	if err == nil || !strings.Contains(err.Error(), "loop detected") {
		t.Error("Failed to detect loop")
	}
}
