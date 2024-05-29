package main

import (
	"context"
	"flag"
	"log"

	"github.com/aaronland/go-indexer"
	"github.com/sfomuseum/go-flags/multi"
	_ "gocloud.dev/blob/fileblob"
)

func main() {

	var bucket_uris multi.MultiString
	var index_uri string

	flag.Var(&bucket_uris, "bucket-uri", "...")
	flag.StringVar(&index_uri, "index-uri", "cwd:///indexer.ix", "")

	flag.Parse()

	ctx := context.Background()

	idx := indexer.NewIndex()
	defer idx.Close()

	err := idx.IndexBuckets(ctx, bucket_uris...)

	if err != nil {
		log.Fatalf("Failed to index buckets, %v", err)
	}

	err = idx.ExportArchiveWithURI(ctx, index_uri)

	if err != nil {
		log.Fatalf("Failed to export index, %v", err)
	}
}
