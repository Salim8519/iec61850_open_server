package runtime

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// renamed are the native keys that would overwrite the bus envelope (coordinator.py:86).
var renamed = map[string]string{"kind": "native_kind", "id": "native_id", "timestamp": "source_timestamp"}

// onWorkerEvent is Coordinator.on_worker_event; it runs on the worker's dispatcher goroutine,
// one event at a time in pipe order, and never waits for an IED actor.
func (c *Coordinator) onWorkerEvent(iedID string, event *pyjson.Object) {
	kind := "native_event"
	if value, ok := event.Get("event"); ok {
		kind, _ = value.(string)
	}
	if kind == "mms_access" { // per-second client summary: live view only (ADR-0066)
		c.access.update(iedID, event)
		return
	}
	if kind == "control_applied" {
		event.Set("external", c.access.control(iedID, event))
		statusRef, hasRef := pyval.Get(event, "status_ref").(string)
		c.pulses.cancel(iedID, statusRef, hasRef, "New MMS control feedback")
		c.mu.Lock()
		item := c.runs[iedID]
		c.mu.Unlock()
		if item != nil {
			if _, known := item.model.index[statusRef]; known {
				timestamp := pyval.Get(event, "timestamp")
				item.table.setValueTime(statusRef, pyval.Get(event, "value"), timestamp)
				if position, ok := item.model.index[statusRef]; ok {
					if timeRef := item.model.signals[position].TimestampRef; timeRef != "" {
						item.table.set(timeRef, timestamp)
						event.Set("updates", pyval.Obj(timeRef, timestamp))
					}
				}
			}
		}
	}
	if kind == "deadband_written" { // a client changed a db/zeroDb the SCD allows (ADR-0118)
		c.mu.Lock()
		item := c.runs[iedID]
		c.mu.Unlock()
		if ref, ok := pyval.Get(event, "ref").(string); ok && item != nil {
			item.table.set(ref, pyval.Get(event, "value"))
		}
	}
	if kind == "worker_failed" {
		c.pulses.cancel(iedID, "", false, "IED worker failed")
		c.mu.Lock()
		c.setIedStateLocked(iedID, "failed")
		c.mu.Unlock()
	}
	payload := pyval.Obj("ied_id", iedID)
	for _, key := range event.Keys() {
		if key == "event" {
			continue
		}
		value, _ := event.Get(key)
		if name, ok := renamed[key]; ok {
			key = name
		}
		payload.Set(key, value)
	}
	c.emit(kind, payload)
	if kind == "control_applied" {
		c.feedback.enqueue(iedID, event)
	}
}
