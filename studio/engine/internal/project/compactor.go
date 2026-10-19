package project

import (
	"sync"
	"unsafe"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Compactor is Compact with memory for the save path (Go engine rule "save the change,
// not the project"). It gives exactly Compact's document, but:
//   - each SCL text is hashed once (a 22 MB project otherwise re-hashes every source on
//     every canvas save), keyed by the text's identity: Go strings are immutable and the
//     cache holds the text, so the address cannot be reused while cached;
//   - when the project's IED list is the very list of the previous call (copy-on-write
//     projects share it between edits), the previous "ieds" and "scl_sources" values are
//     returned again, so the journal and the archive encoder see unchanged values by
//     identity and never encode them to compare.
//
// Projects passed in must be immutable (published copy-on-write trees). Safe for
// concurrent use.
type Compactor struct {
	mu      sync.Mutex
	digests map[stringKey]digestEntry
	list    []any // IED list of the previous call
	ieds    []any
	sources *pyjson.Object
}

type stringKey struct {
	data *byte
	size int
}

type digestEntry struct {
	text   string
	digest string
}

const maxDigests = 64

// NewCompactor returns an empty Compactor.
func NewCompactor() *Compactor { return &Compactor{digests: map[stringKey]digestEntry{}} }

// Compact is document.compact for project.
func (c *Compactor) Compact(project *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range project.Keys() {
		switch key {
		case "ieds", "format", "scl_sources", "migration_notice":
			continue
		}
		value, _ := project.Get(key)
		result.Set(key, value)
	}
	stored, _ := pyval.Lookup(project, "ieds", []any{}).([]any)
	ieds, sources := c.storedIEDs(stored)
	return result.Set("format", int64(CurrentFormat)).Set("ieds", ieds).Set("scl_sources", sources)
}

func (c *Compactor) storedIEDs(stored []any) ([]any, *pyjson.Object) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sources != nil && sameList(stored, c.list) {
		return c.ieds, c.sources
	}
	sources := pyjson.NewObject()
	ieds := []any{}
	used := map[stringKey]bool{}
	for _, item := range stored {
		ied := item.(*pyjson.Object)
		source, _ := ied.Get("source")
		text := source.(string)
		digest := c.digest(text)
		used[keyOf(text)] = true
		sources.Set(digest, source)
		entry := pickFields(ied)
		entry.Set("source_id", digest)
		ieds = append(ieds, entry)
	}
	if len(c.digests) > maxDigests {
		for key := range c.digests {
			if !used[key] {
				delete(c.digests, key)
			}
		}
	}
	c.list, c.ieds, c.sources = stored, ieds, sources
	return ieds, sources
}

// digest is SourceDigest(text), cached by the text's identity. c.mu is held.
func (c *Compactor) digest(text string) string {
	key := keyOf(text)
	if entry, ok := c.digests[key]; ok {
		return entry.digest
	}
	digest := SourceDigest(text)
	c.digests[key] = digestEntry{text: text, digest: digest}
	return digest
}

func keyOf(text string) stringKey { return stringKey{unsafe.StringData(text), len(text)} }

// sameList reports whether a and b are the same list (same backing array and length).
func sameList(a, b []any) bool {
	if len(a) != len(b) {
		return false
	}
	return len(a) == 0 || &a[0] == &b[0]
}
