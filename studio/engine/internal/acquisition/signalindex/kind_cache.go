package signalindex

import (
	"reflect"
	"sync"
)

// kindTable is the kind code of every signal of an index for one data-class map. It keeps the
// map itself, so the map's identity (its version: the route's scl.CachedDataClasses gives a new
// map for a new SCL source) cannot be reused by another map while the table lives.
type kindTable struct {
	classes map[string]string
	codes   []uint8
}

// kindCache remembers the tables of the last two data-class maps: the route asks with the IED's
// classes, and without them for helpers=hide alone. Each table is computed once per index and
// data-class version, so a request only reads one byte per signal (ADR-0113).
type kindCache struct {
	mu     sync.Mutex
	tables [2]*kindTable
}

func sameMap(a, b map[string]string) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() && (a == nil) == (b == nil)
}

// kinds returns the kind codes of every signal for these classes (shared; do not modify).
func (ix *Index) kinds(classes map[string]string) []uint8 {
	cache := &ix.kindCache
	cache.mu.Lock()
	for _, table := range cache.tables {
		if table != nil && sameMap(table.classes, classes) {
			cache.mu.Unlock()
			return table.codes
		}
	}
	cache.mu.Unlock()
	codes := make([]uint8, len(ix.signals))
	for position := range ix.signals {
		codes[position] = kindCode(&ix.signals[position], classes)
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.tables[1] = cache.tables[0]
	cache.tables[0] = &kindTable{classes: classes, codes: codes}
	return codes
}

// needsKinds is true when the query filters, counts or annotates by point kind.
func (q Query) needsKinds() bool {
	return q.PointKind != "" || q.HideHelpers || q.KindCounts || q.Annotate
}

// NeedsClasses is true when the route must look up the IED's data classes: every kind answer
// except helpers=hide alone, whose quality/time helpers never depend on a class.
func (q Query) NeedsClasses() bool {
	return q.PointKind != "" || q.KindCounts || q.Annotate
}
