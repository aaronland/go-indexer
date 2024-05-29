// caisson contains the code used to index
package indexer

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"io/fs"
	"log/slog"
	"sort"
	"strings"
)

type Index struct {
	currentBlockDocumentCount      int
	bloomFilter                    []uint64
	currentDocumentCount           int
	currentBlockStartDocumentCount int
	trigramMethod                  string
	idToFile                       []string
}

type Export struct {
	BloomFilter []uint64 `json:"bloom_filter"`
	IdToFile    []string `json:"id_to_file"`
}

func New() *Index {

	i := &Index{
		currentBlockDocumentCount:      0,
		bloomFilter:                    make([]uint64, 0),
		currentDocumentCount:           0,
		currentBlockStartDocumentCount: 0,
		trigramMethod:                  "default",
		idToFile:                       make([]string, 0),
	}

	return i
}

func (idx *Index) Export(wr io.Writer) error {

	ex := Export{
		BloomFilter: idx.bloomFilter,
		IdToFile:    idx.idToFile,
	}

	enc := json.NewEncoder(wr)
	return enc.Encode(ex)
}

func (idx *Index) Import(r io.Reader) error {

	var ex *Export

	dec := json.NewDecoder(r)
	err := dec.Decode(&ex)

	if err != nil {
		return err
	}

	idx.bloomFilter = ex.BloomFilter
	idx.idToFile = ex.IdToFile

	return nil
}

func (idx *Index) IdToFile(id uint32) string {
	return idx.idToFile[id]
}

func (idx *Index) IndexFS(filesystems ...fs.FS) error {

	for i, this_fs := range filesystems {

		err := idx.indexFS(this_fs)

		if err != nil {
			return fmt.Errorf("Failed to index filesystem at index %d, %w", i, err)
		}
	}

	return nil
}

func (idx *Index) indexFS(this_fs fs.FS) error {

	walk_func := func(path string, info fs.DirEntry, err error) error {

		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil // we only care about files
		}

		res, err := fs.ReadFile(this_fs, path)

		if err != nil {
			slog.Error("Failed to read file", "path", path, "error", err)
			return nil // swallow error
		}

		// don't index binary files by looking for nul byte, similar to how grep does it
		if bytes.IndexByte(res, 0) != -1 {
			return nil
		}

		// only index up to about 5kb
		if len(res) > 5000 {
			res = res[:5000]
		}

		// add the document to the index
		_ = idx.Add(Itemise(idx.Tokenize(string(res))))
		// store the association from what's in the index to the filename, we know its 0 to whatever so this works
		idx.idToFile = append(idx.idToFile, path)

		return nil
	}

	return fs.WalkDir(this_fs, ".", walk_func)
}

// Search the results we need to look at very quickly using only bit operations
// mostly limited by memory access
func (idx *Index) Search(queryBits []uint64) []uint32 {
	var results []uint32
	var res uint64

	if len(queryBits) == 0 {
		return results
	}

	// we want to go through the index, stepping though each "shard"
	for i := 0; i < len(idx.bloomFilter); i += BloomSize {
		// preload the res with the result of the first queryBit and if it's not 0 then we continue
		// if it is 0 it means nothing can be a match so we don't need to do anything
		res = idx.bloomFilter[queryBits[0]+uint64(i)]

		// we don't need to look at the first queryBit anymore so start at one
		// then go through each long looking to see if we keep a match anywhere
		for j := 1; j < len(queryBits); j++ {
			res = res & idx.bloomFilter[queryBits[j]+uint64(i)]

			// if we have 0 meaning no bits set we should bail out because there is nothing more to do here
			// as we cannot have a match even if further queryBits have something set
			if res == 0 {
				break
			}
		}

		// if we have a non 0 value that means at least one bit is set indicating a match
		// so now we need to go through each bit and work out which document it is
		if res != 0 {
			for j := 0; j < DocumentsPerBlock; j++ {
				// determine which bits are still set indicating they have all the bits
				// set for this query which means we have a potential match
				if res&(1<<j) > 0 {
					results = append(results, uint32(DocumentsPerBlock*(i/BloomSize)+j))
				}
			}
		}

	}

	return results
}

// Tokenize returns a slice of tokens for the given text.
func (idx *Index) Tokenize(text string) []string {
	res := strings.Fields(strings.ToLower(text))
	var cres []string
	for _, v := range res {
		if len(v) >= 3 {
			cres = append(cres, v)
		}
	}

	// now we have clean tokens trigram them
	var trigrams []string
	for _, r := range cres {
		switch idx.trigramMethod {
		case "merovius":
			trigrams = append(trigrams, TrigramsMerovius(r)...)
		case "dancantos":
			trigrams = append(trigrams, TrigramsDancantos(r)...)
		case "ffmiruz":
			trigrams = append(trigrams, TrigramsFfmiruz(r)...)
		case "jamesrom":
			for _, t := range TrigramsJamesrom(r) {
				trigrams = append(trigrams, string(t.Bytes()))
			}
		default:
			trigrams = append(trigrams, Trigrams(r)...)
		}

	}

	return trigrams
}

// Itemise given some content will turn it into tokens
// and then use those to create the bit positions we need to
// set for our bloomFilter filter index
func Itemise(tokens []string) []bool {
	docBool := make([]bool, BloomSize)

	for _, token := range tokens {
		for _, i := range HashBloom([]byte(token)) {
			docBool[i] = true
		}
	}
	return docBool
}

// Queryise given some content will turn it into tokens
// and then hash them and store the resulting values into
// a slice which we can use to query the bloom filter
func (idx *Index) Queryise(query string) []uint64 {
	var queryBits []uint64
	for _, w := range idx.Tokenize(query) {
		queryBits = append(queryBits, HashBloom([]byte(w))...)
	}

	// removing duplicates and sorting should in theory improve RAM access
	// and hence performance
	queryBits = RemoveUInt64Duplicates(queryBits)
	sort.Slice(queryBits, func(i, j int) bool {
		return queryBits[i] < queryBits[j]
	})

	return queryBits
}

// Add adds items into the internal bloomFilter used later for pre-screening documents
// note that it fills the filter from right to left, which might not be what you expect
func (idx *Index) Add(item []bool) error {
	// bailout if we ever get something that will break the index
	// because it does not match the size we expect
	if len(item) != BloomSize {
		return errors.New(fmt.Sprintf("expected to match size %d", BloomSize))
	}

	// we need to know if we need to add another batch to this index...
	// which should only be called if we are building from the start
	// or if we need to reset
	if idx.currentBlockDocumentCount == 0 || idx.currentBlockDocumentCount == DocumentsPerBlock {
		idx.bloomFilter = append(idx.bloomFilter, make([]uint64, BloomSize)...)
		idx.currentBlockDocumentCount = 0

		// We don't want to do this for the first document, but everything after
		// we want to know the offset, so in short trail by 1 BloomSize
		if idx.currentDocumentCount != 0 {
			idx.currentBlockStartDocumentCount += BloomSize
		}
	}

	// we need to go through each item and set the correct bit
	for i, bit := range item {
		// if bit is set then we need to flip that bit from its default state, remember this fills from right to left
		// which is not what you expect possibly... anyway it does not matter which way it goes
		if bit {
			idx.bloomFilter[idx.currentBlockStartDocumentCount+i] |= 1 << idx.currentBlockDocumentCount // 0 in this case is the bit we want to flip so it would be 1 if we added document 2 to this block
		}
	}

	// now we increment where we are and where the current block counts are so we can continue to add
	idx.currentBlockDocumentCount++
	idx.currentDocumentCount++

	return nil
}

// PrintIndex prints out the index which can be useful from time
// to time to ensure that bits are being set correctly.
func (idx *Index) PrintIndex() {
	// display what the bloomFilter filter looks like broken into chunks
	for j, i := range idx.bloomFilter {
		if j%BloomSize == 0 {
			fmt.Println("")
		}

		fmt.Printf("%064b\n", i)
	}
}

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
