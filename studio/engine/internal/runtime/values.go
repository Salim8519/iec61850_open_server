package runtime

import (
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// slot is the latest runtime value of one signal and its quality/time companions as the
// Python engine kept them on the signal dict ("value", "quality", "timestamp").
type slot struct {
	value, quality, timestamp any
}

// valueTable holds the values of one IED by point id. Values outlive a run (Python kept
// them on the project's signal dicts until the IED was imported again), so tables are kept
// per IED and rebuilt only when the IED's signal list changes or ResetValues is called.
type valueTable struct {
	mu      sync.RWMutex
	model   *iedModel
	slots   []slot
	origin  *pyjson.Object // the IED object the table was built from
	signals int
}

func newValueTable(model *iedModel) *valueTable {
	table := &valueTable{model: model, slots: make([]slot, len(model.signals)), origin: model.raw,
		signals: len(model.signals)}
	for i, raw := range model.rawSignals {
		table.slots[i] = slot{pyval.Get(raw, "value"), pyval.Get(raw, "quality"), pyval.Get(raw, "timestamp")}
	}
	return table
}

func (t *valueTable) get(ref string) (any, bool) {
	position, ok := t.model.index[ref]
	if !ok {
		return nil, false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.slots[position].value, true
}

func (t *valueTable) set(ref string, value any) {
	if position, ok := t.model.index[ref]; ok {
		t.mu.Lock()
		t.slots[position].value = value
		t.mu.Unlock()
	}
}

func (t *valueTable) setTQ(ref string, timestamp, quality any) {
	if position, ok := t.model.index[ref]; ok {
		t.mu.Lock()
		t.slots[position].timestamp, t.slots[position].quality = timestamp, quality
		t.mu.Unlock()
	}
}

func (t *valueTable) setValueTime(ref string, value, timestamp any) {
	if position, ok := t.model.index[ref]; ok {
		t.mu.Lock()
		t.slots[position].value, t.slots[position].timestamp = value, timestamp
		t.mu.Unlock()
	}
}

// view returns the signal dict with its runtime value, quality and timestamp, keys in the
// project's order (the dict Python returned from Coordinator.signal).
func (t *valueTable) view(ref string) *pyjson.Object {
	position, ok := t.model.index[ref]
	if !ok {
		return nil
	}
	t.mu.RLock()
	current := t.slots[position]
	t.mu.RUnlock()
	raw := t.model.rawSignals[position]
	view := pyjson.NewObject()
	for _, key := range raw.Keys() {
		value, _ := raw.Get(key)
		view.Set(key, value)
	}
	view.Set("value", current.value)
	if current.timestamp != nil || hasKey(raw, "timestamp") {
		view.Set("timestamp", current.timestamp)
	}
	if current.quality != nil || hasKey(raw, "quality") {
		view.Set("quality", current.quality)
	}
	return view
}

func hasKey(object *pyjson.Object, key string) bool {
	_, ok := object.Get(key)
	return ok
}

// tables keeps one valueTable per IED id across runs.
type tables struct {
	mu   sync.Mutex
	byID map[string]*valueTable
}

// forModel returns the IED's table, rebuilding it when the IED's model changed.
func (s *tables) forModel(model *iedModel) *valueTable {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.byID == nil {
		s.byID = map[string]*valueTable{}
	}
	table := s.byID[model.id]
	if table == nil || !sameModel(table, model) {
		table = newValueTable(model)
		s.byID[model.id] = table
	}
	// The same signal list keeps its table (and values) even when the project tree around it
	// was replaced by an unrelated edit.
	return table
}

func sameModel(table *valueTable, model *iedModel) bool {
	if table.origin == model.raw {
		return true
	}
	if table.signals != len(model.signals) {
		return false
	}
	for i, entry := range model.signals {
		if table.model.signals[i] != entry {
			return false
		}
	}
	return true
}

func (s *tables) lookup(id string) *valueTable {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.byID[id]
}

func (s *tables) reset(id string) {
	s.mu.Lock()
	delete(s.byID, id)
	s.mu.Unlock()
}
