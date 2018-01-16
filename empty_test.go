package tiff66

import (
	"encoding/binary"
	"testing"
)

// Create a TIFF node which has several IFDs, but no data fields.
// Check that DeleteEmptyIFDs removes them all and returns nil.
// ifd1 IFD will just have a Next pointer to ifd2
// ifd2 will have sub-IFDs ifd3, ifd4 and ifd5, which will be empty.
func TestEmpty(t *testing.T) {
	node1 := NewIFDNode(TIFFSpace)
	node2 := NewIFDNode(TIFFSpace)
	node3 := NewIFDNode(TIFFSpace)
	node4 := NewIFDNode(TIFFSpace)
	node5 := NewIFDNode(TIFFSpace)
	node1.Order = binary.LittleEndian
	node2.Order = binary.LittleEndian
	node3.Order = binary.LittleEndian
	node4.Order = binary.LittleEndian
	node5.Order = binary.LittleEndian
	node1.Next = node2
	node5size := node5.NodeSize()
	node2.Fields = make([]Field, 2)
	node2.Fields[0] = Field{888, IFD, 2, nil}
	node2.Fields[0].Data = []byte("00000000")
	node2.Fields[1] = Field{999, UNDEFINED, node5size, nil}
	node2.Fields[1].Data = []byte("0000")
	node2.SubIFDs = []SubIFD{{888, node3}, {888, node4}, {999, node5}}
	node1 = node1.DeleteEmptyIFDs()
	if node1 != nil {
		t.Error("DeleteEmptyIFDs didn't return nil.")
	}
}
