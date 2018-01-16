package main

import (
	tiff "github.com/garyhouston/tiff66"
	"io/ioutil"
	"log"
	"os"
)

// Decode a TIFF file, then re-encode it and write to a new file.
func main() {
	logger := log.New(os.Stderr, "", 0)
	if len(os.Args) != 3 {
		logger.Fatalf("Usage: %s file outfile\n", os.Args[0])
	}
	buf, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		logger.Fatal(err)
	}
	valid, order, ifdPos := tiff.GetHeader(buf)
	if !valid {
		logger.Fatal("Not a valid TIFF file")
	}
	root, err := tiff.GetIFDTree(buf, order, ifdPos, tiff.TIFFSpace)
	if err != nil {
		logger.Print(err)
		logger.Print("Error(s) occurred during decoding, but will repack anyway.")
	}
	root.Fix()
	root = root.DeleteEmptyIFDs()
	if root == nil {
		logger.Fatal("Output TIFF file would have no fields; invalid according to TIFF spec.")
	}
	fileSize := tiff.HeaderSize + root.TreeSize()
	out := make([]byte, fileSize)
	tiff.PutHeader(out, order, tiff.HeaderSize)
	next, err := root.PutIFDTree(out, tiff.HeaderSize)
	if err != nil {
		logger.Fatal(err)
	}
	out = out[:next]
	if err = ioutil.WriteFile(os.Args[2], out, 0644); err != nil {
		logger.Fatal(err)
	}
}
