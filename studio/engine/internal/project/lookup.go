package project

import (
	"sync"

	"studio-engine/internal/pyjson"
)

// Lookup answers the "first IED with this id" and "first signal/control with this ref"
// questions of the Python generator expressions (next(item for item in ... if ...)).
// A Lookup from LinearLookup scans like Python; one from an Index answers from maps
// built once per engineering list, which keeps diagram re-validation on every canvas
// save independent of the project's point count. Both give identical answers.
type Lookup struct {
	ieds  []*pyjson.Object
	index *Index
}

// LinearLookup scans ieds on every question.
func LinearLookup(ieds []*pyjson.Object) *Lookup { return &Lookup{ieds: ieds} }

// IED is findByID: the first IED whose id is id, or nil.
func (l *Lookup) IED(id string) *pyjson.Object {
	for _, ied := range l.ieds {
		if text(ied, "id") == id {
			return ied
		}
	}
	return nil
}

// Entry is findByRef over list (an IED's "signals" or "controls"): the first item whose
// ref is ref (nil when none, or when that item is not an object).
func (l *Lookup) Entry(list any, ref string) *pyjson.Object {
	if l.index != nil {
		if entries, ok := list.([]any); ok && len(entries) >= indexedMinimum {
			return l.index.entry(entries, ref)
		}
	}
	return findByRef(list, ref)
}

// indexedMinimum: shorter lists are scanned (a map costs more than a few comparisons).
const indexedMinimum = 32

// Index keeps one ref map per engineering list. Lists are immutable once published
// (copy-on-write project), so a list is identified by its first element's address and
// its length; the entry holds the list itself so the address cannot be reused while
// cached. Safe for concurrent use.
type Index struct {
	mu    sync.Mutex
	lists map[*any]*refIndex
}

type refIndex struct {
	list  []any
	byRef map[string]*pyjson.Object
}

// NewIndex returns an empty index.
func NewIndex() *Index { return &Index{lists: map[*any]*refIndex{}} }

// For returns a Lookup over ieds that uses the index, and forgets the lists of IEDs that
// are no longer in ieds (so removed or replaced models are released).
func (x *Index) For(ieds []*pyjson.Object) *Lookup {
	keep := map[*any]bool{}
	for _, ied := range ieds {
		for _, key := range []string{"signals", "controls"} {
			if list, ok := ied.Get(key); ok {
				if entries, isList := list.([]any); isList && len(entries) > 0 {
					keep[&entries[0]] = true
				}
			}
		}
	}
	x.mu.Lock()
	for key := range x.lists {
		if !keep[key] {
			delete(x.lists, key)
		}
	}
	x.mu.Unlock()
	return &Lookup{ieds: ieds, index: x}
}

func (x *Index) entry(list []any, ref string) *pyjson.Object {
	key := &list[0]
	x.mu.Lock()
	cached := x.lists[key]
	if cached == nil || len(cached.list) != len(list) {
		cached = &refIndex{list: list, byRef: make(map[string]*pyjson.Object, len(list))}
		for _, item := range list {
			entry := object(item)
			name := text(entry, "ref")
			if _, seen := cached.byRef[name]; !seen {
				cached.byRef[name] = entry
			}
		}
		x.lists[key] = cached
	}
	x.mu.Unlock()
	return cached.byRef[ref]
}
