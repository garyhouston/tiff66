package tiff66

import (
	"encoding/binary"
	"testing"
)

// Create a TIFF buffer that has an IFD with a SubIFDs field
// referencing two other IFDs, and check that it's read back
// correctly.
func TestSubIFDs(t *testing.T) {
	node1 := NewIFDNode(TIFFSpace)
	node2 := NewIFDNode(TIFFSpace)
	node3 := NewIFDNode(TIFFSpace)

	node1.Order = binary.LittleEndian
	node1.Fields = make([]Field, 1)
	node1.Fields[0] = Field{SubIFDs, IFD, 2, nil}
	node1.Fields[0].Data = make([]byte, 8)

	node2.Order = node1.Order
	node2.Fields = make([]Field, 1)
	node2.Fields[0] = Field{Compression, SHORT, 1, nil}
	node2.Fields[0].Data = make([]byte, 2)
	node2.Fields[0].PutShort(1, 0, node2.Order)

	node3.Order = node1.Order
	node3.Fields = make([]Field, 1)
	node3.Fields[0] = Field{Compression, SHORT, 1, nil}
	node3.Fields[0].Data = make([]byte, 2)
	node3.Fields[0].PutShort(2, 0, node3.Order)

	node1.SubIFDs = make([]SubIFD, 2)
	node1.SubIFDs[0] = SubIFD{SubIFDs, node2}
	node1.SubIFDs[1] = SubIFD{SubIFDs, node3}

	buf := make([]byte, HeaderSize+node1.TreeSize())
	ifdpos := uint32(HeaderSize)
	PutHeader(buf, node1.Order, ifdpos)
	_, err := node1.PutIFDTree(buf, ifdpos)
	if err != nil {
		t.Error(err)
	}
	valid, getorder, getpos := GetHeader(buf)
	if !valid {
		t.Error("Header not valid")
	}
	if getorder != node1.Order {
		t.Error("Order incorrect")
	}
	if getpos != ifdpos {
		t.Error("Position incorrect")
	}
	getnode1, err := GetIFDTree(buf, getorder, ifdpos, TIFFSpace)
	if err != nil {
		t.Error(err)
	}
	if len(getnode1.SubIFDs) != 2 {
		t.Error("Didn't read back 2 sub-IFDs")
	}
	getnode2 := getnode1.SubIFDs[0].Node
	field2 := getnode2.Fields[0]
	if field2.Tag != Compression {
		t.Error("Wrong tag in first sub-IFD.")
	}
	if field2.Short(0, getnode2.Order) != 1 {
		t.Error("Wrong value in first sub-IFD.")
	}
	getnode3 := getnode1.SubIFDs[1].Node
	field3 := getnode3.Fields[0]
	if field3.Tag != Compression {
		t.Error("Wrong tag in second sub-IFD.")
	}
	if field3.Short(0, getnode3.Order) != 2 {
		t.Error("Wrong value in second sub-IFD.")
	}
}
