package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aaronland/go-indexer"
)

func main() {

	var directory string

	flag.StringVar(&directory, "directory", ".", "The directory to index")
	
	flag.Parse()

	idx := indexer.New()
	err := idx.IndexDirectory(directory)

	if err != nil {
		log.Fatalf("Failed to index directory, %v", err)
	}

	log.Println("done")

	var searchTerm string
	for {
		fmt.Println("enter search term: ")
		_, _ = fmt.Scanln(&searchTerm)

		res := idx.Search(idx.Queryise(searchTerm))
		fmt.Println("--------------")
		fmt.Println(len(res), "index result(s)")
		fmt.Println("")
		
		for _, r := range res {
			fmt.Println(idx.IdToFile(r))
			matching := findMatchingLines(idx.IdToFile(r), searchTerm, 5)
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

// Given a file and a query try to open the file, then look through its lines
// and see if any of them match something from the query up to a limit
// Note this will return partial matches as if any term matches its considered a match
// and there is no accounting for better matches...
// In other words it's a very dumb way of doing this and probably has horrible runtime
// performance to match
func findMatchingLines(filename string, query string, limit int) []string {
	res, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	terms := strings.Fields(strings.ToLower(query))
	var cleanTerms []string
	for _, t := range terms {
		if len(t) >= 3 {
			cleanTerms = append(cleanTerms, t)
		}
	}

	var matches []string
	for i, l := range strings.Split(string(res), "\n") {

		low := strings.ToLower(l)
		found := false
		for _, t := range terms {
			if strings.Contains(low, t) {
				if !found {
					matches = append(matches, fmt.Sprintf("%v. %v", i+1, l))
				}
				found = true
			}
		}

		if len(matches) >= limit {
			return matches
		}
	}

	return matches
}