package indexer

import (
	"hash/fnv"
	"sort"
)

const (
	BloomSize         = 4096
	DocumentsPerBlock = 64
)

// Ngrams given input splits it according the requested size
// such that you can get trigrams or whatever else is required
func Ngrams(text string, size int) []string {
	var runes = []rune(text)

	var ngrams []string

	for i := 0; i < len(runes); i++ {
		if i+size < len(runes)+1 {
			ngram := runes[i : i+size]
			ngrams = append(ngrams, string(ngram))
		}
	}

	return ngrams
}

// GetFill returns the % value of how much this doc was filled, allowing for
// determining if the index will be overfilled for this document
func GetFill(doc []bool) float64 {
	count := 0
	for _, i := range doc {
		if i {
			count++
		}
	}

	return float64(count) / float64(BloomSize) * 100
}

// HashBloom hashes a single token/word 3 times to give us the entry
// locations we need for our bloomFilter filter
func HashBloom(word []byte) []uint64 {
	var hashes []uint64

	h1 := fnv.New64a()
	h2 := fnv.New64()

	// 3 hashes is probably OK for our purposes
	// but to be really like Bing it should change this
	// based on how common/rare the term is where
	// rarer terms are hashes more

	_, _ = h1.Write(word)
	hashes = append(hashes, h1.Sum64()%BloomSize)
	h1.Reset()

	_, _ = h2.Write(word)
	hashes = append(hashes, h2.Sum64()%BloomSize)

	_, _ = h1.Write(word)
	_, _ = h1.Write([]byte("salt")) // anything works here
	hashes = append(hashes, h1.Sum64()%BloomSize)
	h1.Reset()

	return hashes
}

// RemoveUInt64Duplicates removes duplicate values from uint64 slice
func RemoveUInt64Duplicates(s []uint64) []uint64 {
	if len(s) < 2 {
		return s
	}
	sort.Slice(s, func(x, y int) bool { return s[x] > s[y] })
	var e = 1
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			continue
		}
		s[e] = s[i]
		e++
	}

	return s[:e]
}

// runeSize returns rune size given its first byte.
// Returns 1 if invalid first byte.
func runeSize(b byte) int {
	if b < 0xC2 {
		return 1
	}
	if b < 0xE0 {
		return 2
	}
	if b < 0xF0 {
		return 3
	}
	if b < 0xF5 {
		return 4
	}
	return 1
}
