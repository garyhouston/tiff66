package main

import (
	"flag"
	"fmt"
	tiff "github.com/garyhouston/tiff66"
	"io/ioutil"
	"log"
	"os"
)

func printNode(node *tiff.IFDNode, length uint32) {
	fmt.Println()
	fields := node.Fields
	space := node.GetSpace()
	fmt.Printf("%s IFD with %d ", space.Name(), len(fields))
	if len(fields) > 1 {
		fmt.Println("entries:")
	} else {
		fmt.Println("entry:")
	}
	var names map[tiff.Tag]string
	if space == tiff.TIFFSpace {
		names = tiff.TagNames
	}
	for i := 0; i < len(fields); i++ {
		fields[i].Print(node.Order, names, length)
	}
	fmt.Println()
	imageData := node.GetImageData()
	if len(imageData) == 0 {
		fmt.Println("No image data")
	} else {
		fmt.Println("Image data:")
		for _, id := range imageData {
			entry := "entry"
			if len(id.Segments) != 1 {
				entry = "entries"
			}
			fmt.Printf("%s has %d %s, first has length %d\n", tiff.TagNames[id.OffsetTag], len(id.Segments), entry, len(id.Segments[0]))
		}
	}
	for i := 0; i < len(node.SubIFDs); i++ {
		printNode(node.SubIFDs[i].Node, length)
	}
	if node.Next != nil {
		printNode(node.Next, length)
	}
}

// Read and diplay all the IFDs of a TIFF file, including any private IFDs that can be
// detected.
func main() {
	var length uint
	flag.UintVar(&length, "m", 20, "maximum values to print or 0 for no limit")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Printf("Usage: %s [-m max values] file\n", os.Args[0])
		return
	}
	buf, err := ioutil.ReadFile(flag.Arg(0))
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
	printNode(root, uint32(length))
}
