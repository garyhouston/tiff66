package main

import (
	"encoding/binary"
	"fmt"
	tiff "github.com/garyhouston/tiff66"
	"io/ioutil"
	"log"
	"os"
)

func printNode(node *tiff.IFDNode, order binary.ByteOrder) {
	fmt.Println()
	fields := node.IFD.Fields
	fmt.Printf("%s IFD with %d ", node.Space.Name(), len(fields))
	if len(fields) > 1 {
		fmt.Println("entries:")
	} else {
		fmt.Println("entry:")
	}
	var names map[tiff.Tag]string
	if node.Space == tiff.TIFFSpace {
		names = tiff.TagNames
	}
	for i := 0; i < len(fields); i++ {
		fields[i].Print(order, names, 10)
	}
	imageData := node.IFD.ImageData
	fmt.Println()
	if len(imageData) == 0 {
		fmt.Println("No image data")
	} else {
		fmt.Println("Image data:")
		for _, id := range imageData {
			fmt.Printf("%s[0] has length %d\n", tiff.TagNames[id.OffsetField.Tag], len(id.Segments[0]))
		}
	}
	for i := 0; i < len(node.SubIFDs); i++ {
		printNode(node.SubIFDs[i].Node, order)
	}
	if node.Next != nil {
		printNode(node.Next, order)
	}
}

// Read and diplay all the IFDs of a TIFF file, including any private IFDs that can be
// detected.
func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s file\n", os.Args[0])
		return
	}
	buf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	valid, order, ifdPos := tiff.GetHeader(buf)
	if !valid {
		log.Fatal("Not a valid TIFF file")
	}
	root, err := tiff.GetIFDTree(buf, order, ifdPos, tiff.TIFFSpace)
	if err != nil {
		log.Fatal(err)
	}
	printNode(root, order)
}
