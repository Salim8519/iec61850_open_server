package runtime

import (
	"sync"
	"time"
)

// Select-before-operate selections held by the Go engine (ADR-0108). The browser only counts
// down what this answers; Operate checks here whether the selection still holds, so an Operate
// after the window is refused in words instead of reaching an IED whose selection ran out.

// selectionKey names what was selected: an equipment node (point "") or one row of a point panel.
type selectionKey struct{ node, point string }

// selection is one held Select: the control, the value it was selected for and when it ends.
type selection struct {
	iedID, ref string
	action     string // equipment: "open" or "close"; panel rows: ""
	value      any
	options    CommandOptions
	model      string
	windowMs   int
	expires    time.Time
}

// selectionState is what take found.
type selectionState int

const (
	selectionNone selectionState = iota
	selectionLive
	selectionExpired
)

type selections struct {
	mu    sync.Mutex
	now   func() time.Time
	items map[selectionKey]selection
}

func newSelections(now func() time.Time) *selections {
	return &selections{now: now, items: map[selectionKey]selection{}}
}

// hold stores a selection that ends windowMs from now (a newer Select replaces an older one).
func (s *selections) hold(key selectionKey, item selection) selection {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.expires = s.now().Add(time.Duration(item.windowMs) * time.Millisecond)
	s.items[key] = item
	return item
}

// peek reports the selection of key without removing it; an expired one is removed.
func (s *selections) peek(key selectionKey) (selection, selectionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, found := s.items[key]
	if !found {
		return selection{}, selectionNone
	}
	if !s.now().Before(item.expires) {
		delete(s.items, key)
		return item, selectionExpired
	}
	return item, selectionLive
}

// drop removes the selection of key and says whether there was one.
func (s *selections) drop(key selectionKey) (selection, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, found := s.items[key]
	delete(s.items, key)
	return item, found
}

// clear forgets every selection (Stop ends them all).
func (s *selections) clear() {
	s.mu.Lock()
	s.items = map[selectionKey]selection{}
	s.mu.Unlock()
}

// leftMs is how much of the window is left now (0 when it ran out).
func (s *selections) leftMs(item selection) int {
	left := item.expires.Sub(s.now()).Milliseconds()
	if left < 0 {
		return 0
	}
	return int(left)
}
