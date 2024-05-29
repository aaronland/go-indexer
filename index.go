package indexer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"

	"github.com/aaronland/gocloud-blob/bucket"
	"github.com/aaronland/gocloud-blob/walk"
	"gocloud.dev/blob"
)

// Index implements a bloom filter based search index
type Index struct {
	currentBlockDocumentCount      int
	bloomFilter                    []uint64
	currentDocumentCount           int
	currentBlockStartDocumentCount int
	trigramMethod                  string
	idToFile                       []*File
	buckets                        map[string]*blob.Bucket
	bucketURIs                     map[string]int
}

type File struct {
	Path     string `json:"path"`
	BucketId int    `json:"bucket_id`
}

// Archive implements a struct containing data for serializing and deserializing `Index` instances
type Archive struct {
	BloomFilter []uint64       `json:"bloom_filter"`
	IdToFile    []*File        `json:"id_to_file"`
	BucketURIs  map[string]int `json:"bucket_uris"`
}

// NewIndex returns a new (and empty) `Index` instance
func NewIndex() *Index {

	i := &Index{
		currentBlockDocumentCount:      0,
		bloomFilter:                    make([]uint64, 0),
		currentDocumentCount:           0,
		currentBlockStartDocumentCount: 0,
		trigramMethod:                  "default",
		idToFile:                       make([]*File, 0),
		buckets:                        make(map[string]*blob.Bucket),
		bucketURIs:                     make(map[string]int),
	}

	return i
}

func (idx *Index) IndexBuckets(ctx context.Context, bucket_uris ...string) error {

	for i, uri := range bucket_uris {

		b, err := bucket.OpenBucket(ctx, uri)

		if err != nil {
			return fmt.Errorf("Failed to open bucket for '%s', %w", uri, err)
		}

		idx.buckets[uri] = b
		idx.bucketURIs[uri] = i

		err = idx.indexBucket(ctx, b, i)

		if err != nil {
			return fmt.Errorf("Failed to index filesystem at index %d, %w", i, err)
		}
	}

	return nil
}

func (idx *Index) indexBucket(ctx context.Context, b *blob.Bucket, i int) error {

	walk_cb := func(ctx context.Context, obj *blob.ListObject) error {

		if obj.IsDir {
			return nil // we only care about files
		}

		err := idx.IndexObject(ctx, b, i, obj)

		if err != nil {
			return fmt.Errorf("Failed to index %s, %w", obj.Key, err)
		}

		return nil
	}

	return walk.WalkBucket(ctx, b, walk_cb)
}

func (idx *Index) IndexObject(ctx context.Context, b *blob.Bucket, i int, obj *blob.ListObject) error {

	// Replace with NewRangeReader...
	r, err := b.NewReader(ctx, obj.Key, nil)

	if err != nil {
		slog.Warn("Failed to open file for reading", "path", obj.Key, "error", err)
		return nil // swallow error
	}

	defer r.Close()

	res, err := io.ReadAll(r)

	if err != nil {
		slog.Warn("Failed to read file", "path", obj.Key, "error", err)
		return nil
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
	err = idx.Add(Itemise(idx.Tokenize(string(res))))

	if err != nil {
		return err
	}

	// store the association from what's in the index to the filename, we know its 0 to whatever so this works

	f := &File{
		Path:     obj.Key,
		BucketId: i,
	}

	idx.idToFile = append(idx.idToFile, f)
	return nil
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

func (idx *Index) OpenFile(ctx context.Context, id uint32) (io.ReadCloser, error) {

	f := idx.idToFile[id]

	if f == nil {
		return nil, fmt.Errorf("Not found")
	}

	var bucket_uri string

	for uri, idx := range idx.bucketURIs {

		if idx == f.BucketId {
			bucket_uri = uri
			break
		}
	}

	if bucket_uri == "" {
		return nil, fmt.Errorf("Failed to derive bucket URI for file")
	}

	var b *blob.Bucket
	open_b, exists := idx.buckets[bucket_uri]

	if exists {
		b = open_b
	} else {

		new_b, err := bucket.OpenBucket(ctx, bucket_uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to open bucket, %w", err)
		}

		b = new_b
		idx.buckets[bucket_uri] = b
	}

	return b.NewReader(ctx, f.Path, nil)
}

func (idx *Index) IdToFile(id uint32) *File {
	return idx.idToFile[id]
}

func (idx *Index) Archive() *Archive {

	a := &Archive{
		BloomFilter: idx.bloomFilter,
		IdToFile:    idx.idToFile,
		BucketURIs:  idx.bucketURIs,
	}

	return a
}

func (idx *Index) ExportArchiveWithURI(ctx context.Context, archive_uri string) error {

	b, key, err := deriveBucketAndKey(ctx, archive_uri)

	if err != nil {
		return fmt.Errorf("Failed to open bucket (%s) derived from index URI, %w", archive_uri, err)
	}

	defer b.Close()

	wr, err := b.NewWriter(ctx, key, nil)

	if err != nil {
		return fmt.Errorf("Failed to create new writer for archive, %w", err)
	}

	err = idx.ExportArchive(ctx, wr)

	if err != nil {
		return fmt.Errorf("Failed to export archive, %w", err)
	}

	return wr.Close()
}

func (idx *Index) ExportArchive(ctx context.Context, wr io.Writer) error {

	a := idx.Archive()
	enc := json.NewEncoder(wr)
	return enc.Encode(a)
}

func (idx *Index) ImportArchiveWithURI(ctx context.Context, archive_uri string) error {

	b, key, err := deriveBucketAndKey(ctx, archive_uri)

	if err != nil {
		return fmt.Errorf("Failed to open bucket (%s) derived from index URI, %w", archive_uri, err)
	}

	defer b.Close()

	index_r, err := b.NewReader(ctx, key, nil)

	if err != nil {
		return fmt.Errorf("Failed to open index %s for reading, %w", key, err)
	}

	defer index_r.Close()

	return idx.ImportArchive(ctx, index_r)

}

func (idx *Index) ImportArchive(ctx context.Context, r io.Reader) error {

	var a *Archive

	dec := json.NewDecoder(r)
	err := dec.Decode(&a)

	if err != nil {
		return err
	}

	idx.bloomFilter = a.BloomFilter
	idx.idToFile = a.IdToFile
	idx.bucketURIs = a.BucketURIs

	return nil
}

func (idx *Index) Close() error {

	for _, b := range idx.buckets {
		b.Close()
	}

	return nil
}
