package runtime

import (
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// This file exports what the other services need from the coordinator (speed plan S1).
// *Coordinator satisfies projectapi.Runtime and toolsapi.Runtime directly; the gateway and
// acquisition ports, whose method names or slice types collide, are adapted in
// internal/runtime/ports.

// StoppedRefusal is api/context.require_stopped's StudioError text.
const StoppedRefusal = "Stop the simulation before editing its structure"

// CycleLock is coordinator.cycle_lock: Run, Stop, Pause and the cadence tick hold it.
// Lock order: CycleLock -> commandMu -> project mutation -> value tables -> event bus.
func (c *Coordinator) CycleLock() sync.Locker { return &c.lifeMu }

// Emit is coordinator.emit: the event bus plus the run log while a run is open.
func (c *Coordinator) Emit(kind string, payload *pyjson.Object) { c.emit(kind, payload) }

// Publish sends an event to the bus only (never the run log), as mms_access does.
func (c *Coordinator) Publish(kind string, payload *pyjson.Object) { c.publish(kind, payload) }

// RunningIEDs is sorted(coordinator.active).
func (c *Coordinator) RunningIEDs() []string { return c.Running() }

// RunState is coordinator.state (the gateway's name for State).
func (c *Coordinator) RunState() string { return c.State() }

// Stopped is `with cycle_lock: require_stopped(); fn()`: it refuses with Python's
// StudioError unless the simulation is stopped, and runs fn while holding the cycle lock.
func (c *Coordinator) Stopped(fn func() error) error {
	c.lifeMu.Lock()
	defer c.lifeMu.Unlock()
	if c.State() != "stopped" {
		return apperr.New(StoppedRefusal)
	}
	return fn()
}

// RunningSequences counts the scenario runs in state "running" (tools/activity.py).
func (c *Coordinator) RunningSequences() int {
	c.scenarios.mu.Lock()
	defer c.scenarios.mu.Unlock()
	count := 0
	for _, run := range c.scenarios.runs {
		if run.state == "running" {
			count++
		}
	}
	return count
}

// Endpoint is supervisor.endpoint: where a started simulated IED listens; the ValueError
// "IED is not running" when it does not. port is an int64 (Python's int).
func (c *Coordinator) Endpoint(iedID string) (host string, port any, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	point, ok := c.endpoints[iedID]
	if !ok {
		return "", nil, errNotRunning
	}
	return point.host, point.port, nil
}

// Source is coordinator.sources.get((ied_id, ref)): a copy of the run's source spec with its
// current "overridden", nil when the signal has none.
func (c *Coordinator) Source(iedID, ref any) *pyjson.Object {
	ied, okIed := iedID.(string)
	text, okRef := ref.(string)
	if !okIed || !okRef {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.srcs[sourceKey{ied, text}]
	if entry == nil {
		return nil
	}
	spec := pyval.Copy(entry.spec).(*pyjson.Object)
	if entry.overridden != pyval.Truthy(pyval.Get(spec, "overridden")) {
		spec.Set("overridden", entry.overridden)
	}
	return spec
}

// SetIEDState is the runtime-only "state" an acquisition session sets on a real IED
// ("real_live", "real_connecting", "stopped", ...).
func (c *Coordinator) SetIEDState(iedID, state string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setIedStateLocked(iedID, state)
}

// IEDOverlay is the runtime-only part of an IED that Python kept on the project's IED dict:
// "state" ("" when the runtime never set it, so the stored value applies) and
// "goose_state" (present only while HasGoose; Python popped it when the worker stopped).
type IEDOverlay struct {
	State      string
	GooseState any
	HasGoose   bool
}

// IedStates returns the overlay of every IED the runtime knows, for GET /project.
func (c *Coordinator) IedStates() map[string]IEDOverlay {
	c.mu.Lock()
	defer c.mu.Unlock()
	states := make(map[string]IEDOverlay, len(c.iedStates))
	for id, state := range c.iedStates {
		goose, has := c.gooseStates[id]
		states[id] = IEDOverlay{State: state, GooseState: goose, HasGoose: has}
	}
	for id, goose := range c.gooseStates {
		if _, seen := states[id]; !seen {
			states[id] = IEDOverlay{GooseState: goose, HasGoose: true}
		}
	}
	return states
}

// PointKey is an (IED id, reference) pair.
type PointKey struct {
	IED string
	Ref string
}

// PublishChanged is equipment_states.publish_changed for points written outside the
// sample writer (live values).
func (c *Coordinator) PublishChanged(keys []PointKey) {
	converted := make([]bindingKey, len(keys))
	for i, key := range keys {
		converted[i] = bindingKey{key.IED, key.Ref}
	}
	c.plant.publishChanged(converted)
}

// SampleRunning is `ied_id in sample_writer.locks`: a simulated IED whose worker runs.
func (c *Coordinator) SampleRunning(iedID string) bool { return c.simulatedRun(iedID) != nil }

// ApplySamples is sample_writer.apply with quality None; refusals are Python's ValueErrors.
func (c *Coordinator) ApplySamples(iedID string, values []RefValue, source string) error {
	items := make([]item, len(values))
	for i, value := range values {
		items[i] = item{value.Ref, value.Value, nil}
	}
	_, err := c.apply(iedID, items, source)
	return err
}

// SignalRecord is one record of coordinator.index for the gateway (fields of
// gateway/service.Signal): value, the companion references and live_state ("" = absent).
type SignalRecord struct {
	Value        any
	QualityRef   string
	TimestampRef string
	LiveState    string
}

// Record is coordinator.index.get((ied_id, ref)): the point of an IED of the last Run.
func (c *Coordinator) Record(iedID, ref string) (SignalRecord, bool) {
	table := c.indexTable(iedID)
	if table == nil {
		return SignalRecord{}, false
	}
	return table.record(ref)
}
