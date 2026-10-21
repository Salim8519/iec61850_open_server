package runtime

// This file keeps the run's index (Python coordinator.ied_indexes / coordinator.index), the
// run and IED state setters that notify the value hub, and the live-value view acquisition
// writes through (acquisition.Values).

// setStateLocked changes the run state and notifies the hub (Change{}). Needs c.mu.
func (c *Coordinator) setStateLocked(state string) {
	c.state = state
	c.hub.notify(Change{})
}

// ForceState sets the run state without starting or stopping anything: the parity build's
// write-through of the `runtime.state = ...` assignments Python tests make. The product
// never calls it.
func (c *Coordinator) ForceState(state string) {
	c.mu.Lock()
	c.setStateLocked(state)
	c.mu.Unlock()
}

// setIedStateLocked changes an IED's runtime-only "state" and notifies the hub. Needs c.mu.
func (c *Coordinator) setIedStateLocked(iedID, state string) {
	c.iedStates[iedID] = state
	c.hub.notify(Change{IEDID: iedID})
}

// reindexLocked replaces the index with the IEDs of a new run (Coordinator._start). Needs c.mu.
func (c *Coordinator) reindexLocked(items []*running) {
	for id := range c.index {
		delete(c.index, id)
		c.hub.notify(Change{IEDID: id})
	}
	for _, item := range items {
		c.indexLocked(item)
	}
}

// indexLocked adds an IED of the run to the index (run_selection.add_ieds). Needs c.mu.
func (c *Coordinator) indexLocked(item *running) {
	c.index[item.model.id] = item.table
	c.hub.notify(Change{IEDID: item.model.id})
}

// unindexLocked drops an IED from the index (run_selection._forget). Needs c.mu.
func (c *Coordinator) unindexLocked(iedID string) {
	delete(c.index, iedID)
	c.hub.notify(Change{IEDID: iedID})
}

func (c *Coordinator) indexTable(iedID string) *valueTable {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.index[iedID]
}

// RefValue is one point value (the fields of acquisition.RefValue).
type RefValue struct {
	Ref   string
	Value any
}

// Values is the run's value index as acquisition reads and writes it (acquisition.Values;
// internal/runtime/ports adapts the slice type). It covers the IEDs of the last Run and
// keeps their values and live_state after Stop, as Python's ied_indexes did.
type Values struct{ c *Coordinator }

// Values returns the live-value view of the coordinator.
func (c *Coordinator) Values() Values { return Values{c} }

// Has is `ied_id in coordinator.ied_indexes`.
func (v Values) Has(iedID string) bool { return v.c.indexTable(iedID) != nil }

// Get returns a point's value and live_state (nil when never set).
func (v Values) Get(iedID, ref string) (value any, liveState any, ok bool) {
	table := v.c.indexTable(iedID)
	if table == nil {
		return nil, nil, false
	}
	return table.getLive(ref)
}

// SetLive stores device values with live_state "fresh" under one lock; refs the IED does
// not have are ignored.
func (v Values) SetLive(iedID string, values []RefValue) {
	if table := v.c.indexTable(iedID); table != nil {
		table.setLive(values)
	}
}

// MarkStale turns every "fresh" live_state of the IED into "stale".
func (v Values) MarkStale(iedID string) {
	if table := v.c.indexTable(iedID); table != nil {
		table.markStale()
	}
}
