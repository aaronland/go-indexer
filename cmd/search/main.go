package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/aaronland/go-indexer"
	"github.com/sfomuseum/go-flags/multi"
	_ "gocloud.dev/blob/fileblob"
)

func main() {

	var bucket_uris multi.MultiString
	var index_uri string

	flag.Var(&bucket_uris, "bucket-uri", "...")
	flag.StringVar(&index_uri, "index-uri", "", "...")

	flag.Parse()

	ctx := context.Background()

	idx := indexer.NewIndex()
	defer idx.Close()

	if index_uri != "" {

		err := idx.ImportArchiveWithURI(ctx, index_uri)

		if err != nil {
			log.Fatalf("Failed to import index, %v", err)
		}

	} else {

		err := idx.IndexBuckets(ctx, bucket_uris...)

		if err != nil {
			log.Fatalf("Failed to index directory, %v", err)
		}
	}

	var searchTerm string
	for {
		fmt.Println("enter search term: ")
		_, _ = fmt.Scanln(&searchTerm)

		res := idx.Search(idx.Queryise(searchTerm))
		fmt.Println("--------------")
		fmt.Println(len(res), "index result(s)")
		fmt.Println("")

		for _, id := range res {
			fmt.Println(idx.IdToFile(id))

			r, err := idx.OpenFile(ctx, id)

			if err != nil {
				slog.Error("Failed to open file for reading", "id", id, "error", err)
				continue
			}

			defer r.Close()

			matching := indexer.FindMatchingLines(r, searchTerm, 5)

			for _, l := range matching {
				fmt.Println(l)
			}
			if len(matching) == 0 {
				// fmt.Println("false positive match")
			}
			fmt.Println("")
		}
	}

}
