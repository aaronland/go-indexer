package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/aaronland/go-indexer"
	"github.com/sfomuseum/go-flags/multi"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/memblob"
)

func main() {

	var bucket_uris multi.MultiString
	var index string

	flag.Var(&bucket_uris, "bucket-uri", "...")
	flag.StringVar(&index, "index", "", "...")

	flag.Parse()

	ctx := context.Background()

	idx := indexer.NewIndex()
	defer idx.Close()

	if index != "" {

		index_r, err := os.Open(index)

		if err != nil {
			log.Fatalf("Failed to open index, %v", err)
		}

		defer index_r.Close()

		err = idx.Import(index_r)

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
