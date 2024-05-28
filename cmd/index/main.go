package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	
	"github.com/aaronland/go-indexer"
)

func main() {

	var directory string
	var index string
	
	flag.StringVar(&directory, "directory", ".", "The directory to index")
	flag.StringVar(&index, "index", "", "")	
	
	flag.Parse()

	abs_dir, err := filepath.Abs(directory)
	
	if err != nil {
		log.Fatalf("Failed to derive absolute path for directory, %v", err)
	}
	
	idx := indexer.New()
	err = idx.IndexDirectory(abs_dir)

	if err != nil {
		log.Fatalf("Failed to index directory, %v", err)
	}

	if index == "" {

		base := filepath.Base(abs_dir)
		index = fmt.Sprintf("%s.idx", base)
	}
	
	wr, err := os.OpenFile(index, os.O_RDWR|os.O_CREATE, 0600)

	if err != nil {
		log.Fatalf("Failed to open %s for writing, %v", err)
	}

	err = idx.Export(wr)

	if err != nil {
		log.Fatalf("Failed to export index, %v", err)
	}

	err = wr.Close()

	if err != nil {
		log.Fatalf("Failed to close %s after writing, %v", index, err)
	}
}
