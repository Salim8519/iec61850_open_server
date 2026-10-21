package project

import (
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// StoredSource is one distinct original SCL file stored with the project's IEDs
// (projects/scl_sources.stored_sources): id is the first 16 hex digits of its SHA-256.
type StoredSource struct {
	ID       string
	Filename any // ied["filename"] of the first IED loaded from it
	IEDs     []string
	Text     string
}

// sourceIDs is scl_sources.source_id's lru_cache: each text is hashed once, keyed by
// the text's identity (the cache holds the text, so the address is not reused).
var sourceIDs = struct {
	sync.Mutex
	entries map[stringKey]digestEntry
}{entries: map[stringKey]digestEntry{}}

// SourceID is source_id(text).
func SourceID(text string) string {
	key := keyOf(text)
	sourceIDs.Lock()
	defer sourceIDs.Unlock()
	if entry, ok := sourceIDs.entries[key]; ok {
		return entry.digest[:16]
	}
	if len(sourceIDs.entries) >= maxDigests {
		sourceIDs.entries = map[stringKey]digestEntry{} // a project holds at most 20 files
	}
	digest := SourceDigest(text)
	sourceIDs.entries[key] = digestEntry{text: text, digest: digest}
	return digest[:16]
}

// StoredSources is stored_sources(project) as an ordered list (dict insertion order).
func StoredSources(project *pyjson.Object) []StoredSource {
	var result []StoredSource
	index := map[string]int{}
	for _, item := range items(pyval.Lookup(project, "ieds", nil)) {
		ied := object(item)
		source, _ := pyval.Lookup(ied, "source", nil).(string)
		key := SourceID(source)
		position, seen := index[key]
		if !seen {
			position = len(result)
			index[key] = position
			result = append(result, StoredSource{ID: key, Filename: pyval.Lookup(ied, "filename", nil), Text: source})
		}
		result[position].IEDs = append(result[position].IEDs, text(ied, "name"))
	}
	return result
}

// RequireSource is scl_sources.require_source.
func RequireSource(project *pyjson.Object, key string) (StoredSource, error) {
	for _, entry := range StoredSources(project) {
		if entry.ID == key {
			return entry, nil
		}
	}
	return StoredSource{}, apperr.Coded("This SCL file is not in the project", "not_found")
}

// IEDNames is entry["ieds"] as a JSON list.
func (s StoredSource) IEDNames() []any {
	names := make([]any, len(s.IEDs))
	for index, name := range s.IEDs {
		names[index] = name
	}
	return names
}
