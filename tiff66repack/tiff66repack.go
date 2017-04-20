package main

import (
	"fmt"
	tiff "github.com/garyhouston/tiff66"
	"io/ioutil"
	"log"
	"os"
)

// Decode a TIFF file, then re-encode it and write to a new file.
func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s file outfile\n", os.Args[0])
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
	root.Fix(order)
	fileSize := tiff.HeaderSize + root.TreeSize()
	out := make([]byte, fileSize)
	tiff.PutHeader(out, order, tiff.HeaderSize)
	next, err := root.PutIFDTree(out, tiff.HeaderSize, order)
	if err != nil {
		log.Fatal(err)
	}
	out = out[:next]
	ioutil.WriteFile(os.Args[2], out, 0644)
}
