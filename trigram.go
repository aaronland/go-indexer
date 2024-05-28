package indexer

import (
	"unicode/utf8"
)

// Trigrams takes in text and returns its trigrams
// Attempts to be as efficient as possible
func Trigrams(text string) []string {
	var runes = []rune(text)

	// if we have less than or 2 runes we cannot do anything so bail out
	if len(runes) <= 2 {
		return []string{}
	}

	// we always need this many ngrams, so preallocate to avoid expanding the slice
	// which is the most expensive thing in here according to profiles
	ngrams := make([]string, len(runes)-2)

	for i := 0; i < len(runes); i++ {
		if i+3 < len(runes)+1 {
			ngram := runes[i : i+3]
			ngrams[i] = string(ngram)
		}
	}

	return ngrams
}

func TrigramsMerovius(text string) []string {
	var offsets [3]int
	for i := 0; i < 2; i++ {
		if len(text) <= offsets[i] {
			return nil
		}
		r, sz := utf8.DecodeRuneInString(text[offsets[i]:])
		if r == utf8.RuneError {
			return nil
		}
		offsets[i+1] = offsets[i] + sz
	}
	out := make([]string, 0, len(text))
	for offsets[2] < len(text) {
		r, sz := utf8.DecodeRuneInString(text)
		if r == utf8.RuneError {
			return nil
		}
		out = append(out, text[offsets[0]:offsets[2]+sz])
		copy(offsets[:2], offsets[1:])
		offsets[2] += sz
	}
	return out
}

// Trigrams takes in text and returns its trigrams
func TrigramsDancantos(text string) []string {
	if len(text) < 3 {
		return []string{}
	}
	result := make([]string, len(text)-2)
	trigramsDancantos(text, result, 0)
	return result
}

// trigramsDancantos takes in text and inserts its trigrams to the result slice starting at location.
// Attempts to be as efficient as possible
func trigramsDancantos(text string, result []string, location int) int {
	l := len(text)
	if l < 3 {
		return 0
	}

	// set up vars to track locations in the text as we walk
	st, mid, end, tmp := 0, 0, 0, 0

	// mid = index of second rune
	// end = index of third rune
	_, mid = utf8.DecodeRuneInString(text)
	_, tmp = utf8.DecodeRuneInString(text[mid:])
	end = mid + tmp
	for end < l {
		result[location] = text[st : end+1]
		_, tmp = utf8.DecodeRuneInString(text[end:])
		// update start, mid, end = old+mid, old_end, new_end=old_end+tmp
		st, mid, end = mid, end, end+tmp
		location++
	}
	return len(text) - 2
}

type Trigram [3]rune

// Bytes is the simplest way to turn an array of runes into a slice of bytes.
// There is a faster way to do this, but not needed for this demo.
// See: https://stackoverflow.com/questions/29255746/how-encode-rune-into-byte-using-utf8
func (t Trigram) Bytes() []byte {
	return []byte(string(t[:]))
}

// Trigrams takes in text and returns its trigrams.
func TrigramsJamesrom(text string) []Trigram {
	runes := []rune(text)

	// if we have less than or 2 runes we cannot do anything so bail out
	if len(runes) < 3 {
		return []Trigram{}
	}

	// allocate all trigrams
	ngrams := make([]Trigram, len(runes)-2)

	// create the trigrams
	for i := 0; i < len(runes)-2; i++ {
		ngrams[i] = Trigram(runes[i : i+3])
	}

	return ngrams
}

func TrigramsFfmiruz(text string) []string {
	var gram [3]int
	for i := 0; i < 2; i++ {
		size := runeSize((text[gram[i]]))
		gram[i+1] = gram[i] + size
	}

	list := make([]string, 0, len(text))
	for gram[2] < len(text) {
		size := runeSize((text[gram[2]]))
		list = append(list, text[gram[0]:gram[2]+size])
		gram[0], gram[1], gram[2] = gram[1], gram[2], gram[2]+size
	}
	return list
}
