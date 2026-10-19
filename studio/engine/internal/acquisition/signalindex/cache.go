package signalindex

import (
	"sync"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Cache keeps one Index per IED. An entry is valid only for the exact model object it was
// built from: a new model object (a re-import or any project change that replaces the IED)
// rebuilds it. Entries are keyed by the IED "id", so replaced models do not pile up. Safe
// for concurrent use; no lock is held while an index is built.
type Cache struct {
	mu      sync.Mutex
	entries map[string]cacheEntry
}

type cacheEntry struct {
	ied   *pyjson.Object
	index *Index
}

// For returns the index of ied, building it when the cached one belongs to another object.
func (c *Cache) For(ied *pyjson.Object) *Index {
	key := pyv.PyStr(pyv.Get(ied, "id"))
	c.mu.Lock()
	entry, found := c.entries[key]
	c.mu.Unlock()
	if found && entry.ied == ied {
		return entry.index
	}
	index := Build(ied)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.entries == nil {
		c.entries = map[string]cacheEntry{}
	}
	c.entries[key] = cacheEntry{ied: ied, index: index}
	return index
}

// Forget drops the index of an IED id (for example after the IED was deleted).
func (c *Cache) Forget(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, id)
}
