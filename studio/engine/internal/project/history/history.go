// Package history is the project's Undo/Redo memory (ADR-0135). It is pure: it keeps, for
// each change, the top-level project values before and after, and answers which values to
// put back. It never writes, publishes or locks; the project service calls it under its
// mutation lock and commits what it answers like any other change.
//
// Project trees are copy-on-write, so a step holds references into immutable trees: a step
// costs only the parts that really changed. At most Limit steps are kept, in memory only.
package history

import (
	"time"

	"studio-engine/internal/pyjson"
)

// Limit is how many steps Undo can go back (owner choice, 2026-11-02).
const Limit = 20

// Window joins quick edits of the same thing from the same place (typing a label, dragging
// a symbol) into one step.
const Window = time.Second

// Missing stands for a key the project did not have.
var Missing = missing{}

type missing struct{}

// Rules are the project knowledge the history needs. Automatic changes (addresses placed on
// a network card, the gateway's own on/off memory) are not steps: Same ignores them and Keep
// carries their current value over when a step is undone or redone.
type Rules interface {
	// Skip says a top-level key is never part of a step (its identity, this PC's network).
	Skip(key string) bool
	// Same says two values of key are equal for Undo (automatic parts ignored).
	Same(key string, a, b any) bool
	// Keep returns restored with the automatic parts of current.
	Keep(key string, restored, current any) any
}

type step struct {
	keys          []string
	before, after []any
	source        any
	at            time.Time
}

// History is the Undo and Redo stacks of one open project.
type History struct {
	rules      Rules
	now        func() time.Time
	undo, redo []step
}

// New returns an empty history.
func New(rules Rules, now func() time.Time) *History {
	return &History{rules: rules, now: now}
}

// Record adds the change from prev to next as a step and forgets the Redo steps. A change
// that only touched skipped or automatic parts is no step; a quick follow-up of the last
// step (same keys and source, within Window, continuing from its result) joins it.
func (h *History) Record(prev, next *pyjson.Object, source any) {
	keys := h.changed(prev, next)
	if len(keys) == 0 {
		return
	}
	added := step{keys: keys, before: values(prev, keys), after: values(next, keys), source: source, at: h.now()}
	h.redo = nil
	if last := len(h.undo) - 1; last >= 0 && h.joins(h.undo[last], added) {
		h.undo[last].after, h.undo[last].at = added.after, added.at
		return
	}
	h.undo = append(h.undo, added)
	if len(h.undo) > Limit {
		h.undo = append([]step(nil), h.undo[len(h.undo)-Limit:]...) // release the oldest step
	}
}

// Clear forgets every step (another project was opened, created or closed).
func (h *History) Clear() { h.undo, h.redo = nil, nil }

// Plan is what Undo (or Redo) puts back: Keys and their Values, Missing meaning "remove".
type Plan struct {
	Keys   []string
	Values []any
}

// Next answers the plan for one Undo (undo=true) or Redo step against the current project,
// or false when there is none. A step whose values were changed since by something that is
// not a step can never apply again: that stack is dropped, together with the older steps
// that depend on it.
func (h *History) Next(undo bool, current *pyjson.Object) (Plan, bool) {
	stack := h.stack(undo)
	if len(*stack) == 0 {
		return Plan{}, false
	}
	top := (*stack)[len(*stack)-1]
	expected, target := top.after, top.before
	if !undo {
		expected, target = top.before, top.after
	}
	plan := Plan{Keys: top.keys, Values: make([]any, len(top.keys))}
	for index, key := range top.keys {
		now := value(current, key)
		if !h.rules.Same(key, now, expected[index]) {
			*stack = nil
			return Plan{}, false
		}
		plan.Values[index] = h.rules.Keep(key, target[index], now)
	}
	return plan, true
}

// Done moves the step Next answered to the other stack once its plan is committed.
func (h *History) Done(undo bool) {
	from, to := h.stack(undo), h.stack(!undo)
	last := len(*from) - 1
	moved := (*from)[last]
	moved.at = time.Time{} // a stepped change never joins a later edit
	*from = (*from)[:last]
	*to = append(*to, moved)
}

// Keys are the top-level keys of the next Undo or Redo step (nil when there is none).
func (h *History) Keys(undo bool) []string {
	stack := *h.stack(undo)
	if len(stack) == 0 {
		return nil
	}
	return stack[len(stack)-1].keys
}

func (h *History) stack(undo bool) *[]step {
	if undo {
		return &h.undo
	}
	return &h.redo
}

func (h *History) joins(last, added step) bool {
	if last.at.IsZero() || added.at.Sub(last.at) > Window || !Identical(last.source, added.source) ||
		len(last.keys) != len(added.keys) {
		return false
	}
	for index, key := range last.keys {
		if key != added.keys[index] || !h.rules.Same(key, last.after[index], added.before[index]) {
			return false
		}
	}
	return true
}

// changed lists the keys of prev and next (in that order) whose values differ.
func (h *History) changed(prev, next *pyjson.Object) []string {
	var keys []string
	seen := map[string]bool{}
	for _, source := range []*pyjson.Object{prev, next} {
		for _, key := range source.Keys() {
			if seen[key] || h.rules.Skip(key) {
				continue
			}
			seen[key] = true
			if !h.rules.Same(key, value(prev, key), value(next, key)) {
				keys = append(keys, key)
			}
		}
	}
	return keys
}

func value(project *pyjson.Object, key string) any {
	if found, ok := project.Get(key); ok {
		return found
	}
	return Missing
}

func values(project *pyjson.Object, keys []string) []any {
	out := make([]any, len(keys))
	for index, key := range keys {
		out[index] = value(project, key)
	}
	return out
}
