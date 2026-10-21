package xsd

import (
	"container/list"

	"studio-engine/internal/scl"
)

// lru keeps the most recently used verdicts in memory; it is not safe for concurrent
// use (the cache guards it).
type lru struct {
	limit int
	order *list.List // front = most recent; values are *lruEntry
	index map[string]*list.Element
}

type lruEntry struct {
	key     string
	verdict scl.SchemaVerdict
}

func newLRU(limit int) *lru {
	if limit < 1 {
		limit = 1
	}
	return &lru{limit: limit, order: list.New(), index: map[string]*list.Element{}}
}

func (l *lru) get(key string) (scl.SchemaVerdict, bool) {
	element, found := l.index[key]
	if !found {
		return scl.SchemaVerdict{}, false
	}
	l.order.MoveToFront(element)
	return copyVerdict(element.Value.(*lruEntry).verdict), true
}

func (l *lru) put(key string, verdict scl.SchemaVerdict) {
	if element, found := l.index[key]; found {
		element.Value.(*lruEntry).verdict = copyVerdict(verdict)
		l.order.MoveToFront(element)
		return
	}
	l.index[key] = l.order.PushFront(&lruEntry{key: key, verdict: copyVerdict(verdict)})
	for l.order.Len() > l.limit {
		oldest := l.order.Back()
		l.order.Remove(oldest)
		delete(l.index, oldest.Value.(*lruEntry).key)
	}
}

func (l *lru) len() int { return l.order.Len() }

// copyVerdict keeps cached verdicts safe from callers that modify the returned errors.
func copyVerdict(verdict scl.SchemaVerdict) scl.SchemaVerdict {
	errors := make([]scl.SchemaError, len(verdict.Errors))
	copy(errors, verdict.Errors)
	return scl.SchemaVerdict{Status: verdict.Status, Errors: errors}
}
