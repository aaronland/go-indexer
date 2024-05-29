package indexer

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Given a file and a query try to open the file, then look through its lines
// and see if any of them match something from the query up to a limit
// Note this will return partial matches as if any term matches its considered a match
// and there is no accounting for better matches...
// In other words it's a very dumb way of doing this and probably has horrible runtime
// performance to match
func FindMatchingLines(r io.Reader, query string, limit int) []string {

	var matches []string

	res, err := io.ReadAll(r)

	if err != nil {
		slog.Error("Failed to read body to match lines", "error", err)
		return matches
	}

	terms := strings.Fields(strings.ToLower(query))
	var cleanTerms []string
	for _, t := range terms {
		if len(t) >= 3 {
			cleanTerms = append(cleanTerms, t)
		}
	}

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
