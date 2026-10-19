// Package archive reads and writes .subsim project archives exactly as
// studio_app.projects.archive does: a ZIP with the single deflated member project.json holding
// json.dumps(compact(project), allow_nan=False) (format 15).
//
// Writing is incremental (Go engine rule "save the change, not the project"): project.json
// is cut into segments - one per top-level key and one per scl_sources text - and every
// segment is deflated on its own, ending with a sync flush, so the compressed segments
// concatenate into one valid deflate stream. An Encoder caches the compressed bytes, length
// and CRC-32 of each SCL text by its digest and of each other segment by its exact bytes, and
// combines CRCs with zlib's crc32_combine, so a save after a diagram edit re-deflates only
// the changed keys. The result is byte-identical JSON to pyjson.Dumps(doc) and a ZIP that
// Python's zipfile reads and verifies.
package archive

import (
	"bytes"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// MaxUnpacked is MAX_UNPACKED: the save limit and the compressed archive limit (100 MiB).
const MaxUnpacked = 100 * 1024 * 1024

// MaxLegacyUnpacked is MAX_LEGACY_UNPACKED (256 MiB) for old inline-source archives.
const MaxLegacyUnpacked = 256 * 1024 * 1024

const sclKey = "scl_sources"

// Encoder writes archives and keeps the compressed segments of the last documents.
// It is safe for concurrent use; documents passed to it must not be mutated afterwards.
type Encoder struct {
	// Now gives the ZIP timestamp (local time, like time.localtime()); nil means time.Now.
	Now func() time.Time

	mu       sync.Mutex
	sources  map[string]*sourceEntry
	literals map[string]*segment
}

type sourceEntry struct {
	text string
	seg  *segment
}

const maxCachedSources = 40 // two projects of 20 sources

// NewEncoder returns an Encoder with an empty segment cache.
func NewEncoder() *Encoder {
	return &Encoder{sources: map[string]*sourceEntry{}, literals: map[string]*segment{}}
}

// Encode is encode(project) for an already compacted document, without a cache.
func Encode(compactDoc *pyjson.Object) ([]byte, error) {
	return NewEncoder().Encode(compactDoc)
}

// Encode returns the archive bytes of compactDoc. Errors: Python's ValueError "Out of range
// float values are not JSON compliant: nan" (inf, -inf) for a non-finite float, and the
// StudioError "Engineering project exceeds the 100 MB save limit; nothing was written".
func (e *Encoder) Encode(compactDoc *pyjson.Object) ([]byte, error) {
	segments, crc, size, err := e.prepare(compactDoc)
	if err != nil {
		return nil, err
	}
	now := time.Now
	if e.Now != nil {
		now = e.Now
	}
	return assemble(compressAll(segments), crc, size, now()), nil
}

// Checksum returns the CRC-32 and length of project.json for compactDoc without compressing
// anything (the journal compares them with the autosave's central directory).
func (e *Encoder) Checksum(compactDoc *pyjson.Object) (uint32, int64, error) {
	_, crc, size, err := e.prepare(compactDoc)
	return crc, size, err
}

// Warm compresses and caches every segment of compactDoc (for example right after a project
// was opened), so the first save afterwards is fast.
func (e *Encoder) Warm(compactDoc *pyjson.Object) error {
	segments, _, _, err := e.prepare(compactDoc)
	if err == nil {
		compressAll(segments)
	}
	return err
}

func (e *Encoder) prepare(doc *pyjson.Object) ([]*segment, uint32, int64, error) {
	segments, err := e.plan(doc)
	if err != nil {
		return nil, 0, 0, err
	}
	crc, size := combine(segments)
	if size > MaxUnpacked {
		return nil, 0, 0, apperr.New("Engineering project exceeds the 100 MB save limit; nothing was written")
	}
	return segments, crc, size, nil
}

// plan cuts json.dumps(doc) into segments, reusing cached ones.
func (e *Encoder) plan(doc *pyjson.Object) ([]*segment, error) {
	var segments []*segment
	used := map[string]bool{}
	pending := []byte{'{'}
	flush := func(slot string) {
		segments = append(segments, e.literal(slot, pending))
		pending = nil
	}
	for index, key := range doc.Keys() {
		if index > 0 {
			pending = append(pending, ", "...)
		}
		pending = append(pyjson.AppendString(pending, key), ": "...)
		value, _ := doc.Get(key)
		if sources, ok := sourceTable(key, value); ok {
			pending = append(pending, '{')
			for position, digest := range sources.Keys() {
				if position > 0 {
					pending = append(pending, ", "...)
				}
				pending = append(pyjson.AppendString(pending, digest), ": "...)
				flush(sclKey + "\x00" + digest)
				text, _ := sources.Get(digest)
				segments = append(segments, e.source(digest, text.(string)))
				used[digest] = true
			}
			pending = append(pending, '}')
			continue
		}
		if err := checkFinite(value); err != nil {
			return nil, err
		}
		encoded, err := pyjson.Dumps(value)
		if err != nil {
			return nil, err
		}
		pending = append(pending, encoded...)
		flush(key)
	}
	pending = append(pending, '}')
	flush("\x00end")
	e.evict(used)
	return segments, nil
}

// sourceTable reports whether value is the scl_sources table of string texts.
func sourceTable(key string, value any) (*pyjson.Object, bool) {
	table, ok := value.(*pyjson.Object)
	if key != sclKey || !ok {
		return nil, false
	}
	for _, digest := range table.Keys() {
		if text, _ := table.Get(digest); !isString(text) {
			return nil, false
		}
	}
	return table, true
}

func isString(value any) bool {
	_, ok := value.(string)
	return ok
}

func (e *Encoder) literal(slot string, raw []byte) *segment {
	e.mu.Lock()
	defer e.mu.Unlock()
	if cached := e.literals[slot]; cached != nil && bytes.Equal(cached.raw, raw) {
		return cached
	}
	created := newSegment(raw, true) // plan hands over raw: it starts a new buffer after a flush
	e.literals[slot] = created
	return created
}

func (e *Encoder) source(digest, text string) *segment {
	e.mu.Lock()
	cached := e.sources[digest]
	e.mu.Unlock()
	if cached != nil && cached.text == text { // same backing string: no byte comparison
		return cached.seg
	}
	created := &sourceEntry{text: text, seg: newSegment(pyjson.AppendString(nil, text), false)}
	e.mu.Lock()
	e.sources[digest] = created
	e.mu.Unlock()
	return created.seg
}

func (e *Encoder) evict(used map[string]bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.sources) > maxCachedSources {
		for digest := range e.sources {
			if !used[digest] {
				delete(e.sources, digest)
			}
		}
	}
	if len(e.literals) > 4*maxCachedSources {
		e.literals = map[string]*segment{}
	}
}
