package signalindex

import (
	"sort"
	"strings"
)

// textIndex holds every casefolded haystack in one string, each followed by a NUL
// separator, so a q filter is one fast substring scan that jumps from match to match
// instead of a search per signal.
type textIndex struct {
	corpus string
	starts []int // starts[i] is where signal i's haystack begins in corpus
}

// newTextIndex joins the haystacks and returns the index with each haystack as a slice of
// the shared corpus (no second copy).
func newTextIndex(haystacks []string) (textIndex, []string) {
	size := 0
	for _, text := range haystacks {
		size += len(text) + 1
	}
	var corpus strings.Builder
	corpus.Grow(size)
	starts := make([]int, len(haystacks))
	for index, text := range haystacks {
		starts[index] = corpus.Len()
		corpus.WriteString(text)
		corpus.WriteByte(0)
	}
	index := textIndex{corpus: corpus.String(), starts: starts}
	views := make([]string, len(haystacks))
	for position, start := range starts {
		views[position] = index.corpus[start : start+len(haystacks[position])]
	}
	return index, views
}

// matches returns the positions of the haystacks containing needle, in order, and false
// when needle holds a NUL (a match could then cross a separator; the caller scans instead).
// needle must not be empty.
func (t textIndex) matches(needle string) ([]int, bool) {
	if strings.IndexByte(needle, 0) >= 0 {
		return nil, false
	}
	var positions []int
	for from := 0; from < len(t.corpus); {
		found := strings.Index(t.corpus[from:], needle)
		if found < 0 {
			break
		}
		at := from + found
		position := sort.Search(len(t.starts), func(i int) bool { return t.starts[i] > at }) - 1
		positions = append(positions, position)
		next := len(t.corpus)
		if position+1 < len(t.starts) {
			next = t.starts[position+1]
		}
		from = next
	}
	return positions, true
}
