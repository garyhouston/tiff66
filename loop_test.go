package tiff66

import (
	"encoding/binary"
	"strings"
	"testing"
)

// Create a TIFF buffer where one IFD refers to another and vice versa, and
// check that reading it back gives an error.
func TestLoop(t *testing.T) {
	node1 := NewIFDNode(TIFFSpace)
	node2 := NewIFDNode(TIFFSpace)
	node1.Order = binary.LittleEndian
	node1.Fields = make([]Field, 1)
	node1.Fields[0] = Field{Compression, BYTE, 1, nil}
	node1.Fields[0].Data = []byte("\001")
	node2.Order = node1.Order
	node2.Fields = make([]Field, 1)
	node2.Fields[0] = node1.Fields[0]
	ifdsize := node1.TableSize() // ifd contains no external data
	buf := make([]byte, HeaderSize+2*ifdsize)
	ifd1pos := uint32(HeaderSize)
	ifd2pos := ifd1pos + ifdsize
	PutHeader(buf, node1.Order, ifd1pos)
	_, err := node1.put(buf, ifd1pos, nil, ifd2pos)
	if err != nil {
		t.Error("Failed to put ifd1")
	}
	_, err = node2.put(buf, ifd2pos, nil, ifd1pos)
	if err != nil {
		t.Error("Failed to put ifd2")
	}
	valid, getorder, getpos := GetHeader(buf)
	if !valid {
		t.Error("Header not valid")
	}
	_, err = GetIFDTree(buf, getorder, getpos, TIFFSpace)
	if err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Error("Failed to detect cycle")
	}
}
