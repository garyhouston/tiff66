package tiff66

import (
	"encoding/binary"
	"testing"
)

// Create a TIFF buffer that has an IFD with a SubIFDs field
// referencing two other IFDs, and check that it's read back
// correctly.
func TestSubIFDs(t *testing.T) {
	order := binary.LittleEndian
	var ifd1, ifd2, ifd3 IFD_T
	ifd1.Fields = make([]Field, 1)
	ifd1.Fields[0] = Field{SubIFDs, IFD, 2, nil}
	ifd1.Fields[0].Data = make([]byte, 8)

	ifd2.Fields = make([]Field, 1)
	ifd2.Fields[0] = Field{Compression, SHORT, 1, nil}
	ifd2.Fields[0].Data = make([]byte, 2)
	ifd2.Fields[0].PutShort(1, 0, order)

	ifd3.Fields = make([]Field, 1)
	ifd3.Fields[0] = Field{Compression, SHORT, 1, nil}
	ifd3.Fields[0].Data = make([]byte, 2)
	ifd3.Fields[0].PutShort(2, 0, order)

	node1 := IFDNode{ifd1, TIFFSpace, nil, nil}
	node2 := IFDNode{ifd2, TIFFSpace, nil, nil}
	node3 := IFDNode{ifd3, TIFFSpace, nil, nil}

	node1.SubIFDs = make([]SubIFD, 2)
	node1.SubIFDs[0] = SubIFD{SubIFDs, &node2}
	node1.SubIFDs[1] = SubIFD{SubIFDs, &node3}

	buf := make([]byte, HeaderSize()+node1.TreeSize())
	ifdpos := HeaderSize()
	PutHeader(buf, order, ifdpos)
	_, err := node1.PutIFDTree(buf, ifdpos, order)
	if err != nil {
		t.Error(err)
	}
	valid, getorder, getpos := GetHeader(buf)
	if !valid {
		t.Error("Header not valid")
	}
	if getorder != order {
		t.Error("Order incorrect")
	}
	if getpos != ifdpos {
		t.Error("Position incorrect")
	}
	getnode1, err := GetIFDTree(buf, order, ifdpos, TIFFSpace)
	if err != nil {
		t.Error(err)
	}
	if len(getnode1.SubIFDs) != 2 {
		t.Error("Didn't read back 2 sub-IFDs")
	}
	getnode2 := getnode1.SubIFDs[0].Node
	field2 := getnode2.IFD.Fields[0]
	if field2.Tag != Compression {
		t.Error("Wrong tag in first sub-IFD.")
	}
	if field2.Short(0, order) != 1 {
		t.Error("Wrong value in first sub-IFD.")
	}
	getnode3 := getnode1.SubIFDs[1].Node
	field3 := getnode3.IFD.Fields[0]
	if field3.Tag != Compression {
		t.Error("Wrong tag in second sub-IFD.")
	}
	if field3.Short(0, order) != 2 {
		t.Error("Wrong value in second sub-IFD.")
	}
}
