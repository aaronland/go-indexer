package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/aaronland/go-indexer"
	"github.com/sfomuseum/go-flags/multi"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/memblob"
)

func main() {

	var bucket_uris multi.MultiString
	var index string

	flag.Var(&bucket_uris, "bucket-uri", "...")
	flag.StringVar(&index, "index", "", "")

	flag.Parse()

	ctx := context.Background()

	idx := indexer.NewIndex()
	defer idx.Close()

	err := idx.IndexBuckets(ctx, bucket_uris...)

	if err != nil {
		log.Fatalf("Failed to index buckets, %v", err)
	}

	if index == "" {
		index = "indexer.idx"
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
