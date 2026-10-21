package acquisition

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

var conflictKeys = []string{"ied_id", "gocb_ref", "src_mac", "sharing"}

// onEvent handles studio_goose events (goose_engine.py:102-125). It takes only mu, so a
// request waiting under calls never blocks event delivery.
func (e *GooseEngine) onEvent(event *pyjson.Object) {
	emit := e.deps.Events.Emit
	switch pyv.GetStr(event, "event") {
	case "goose_conflict":
		payload := pyjson.NewObject()
		for _, key := range conflictKeys {
			payload.Set(key, pyv.Get(event, key))
		}
		id := pyv.PyStr(pyv.Get(event, "ied_id"))
		e.mu.Lock()
		previous := pyv.GetObj(e.conflicts, id)
		entry := pyv.Shallow(payload)
		entry.Set("choice", pyv.Get(previous, "choice"))
		e.conflicts.Set(id, entry)
		e.mu.Unlock()
		emit("goose_conflict", payload)
	case "goose_share":
		e.mu.Lock()
		e.link.Set("share", pyv.GetOr(event, "state", "off"))
		state := pyv.Get(e.link, "share")
		e.mu.Unlock()
		emit("goose_share", pyv.O("state", state))
	case "goose_uplink":
		e.mu.Lock()
		e.link.Set("uplink", pyv.GetOr(event, "state", ""))
		e.link.Set("message", pyv.GetOr(event, "message", ""))
		e.mu.Unlock()
		emit("goose_uplink", pyv.O("state", pyv.Get(event, "state"), "message", pyv.GetOr(event, "message", "")))
	case "goose_storm":
		emit("goose_storm", pyv.O("ied_id", pyv.Get(event, "ied_id"),
			"limit_per_second", pyv.Get(event, "limit_per_second")))
	case "worker_failed":
		e.mu.Lock()
		e.link = closedLink()
		e.link.Set("uplink", "failed")
		e.link.Set("message", "The GOOSE engine stopped unexpectedly")
		e.mu.Unlock()
		message := pyv.Get(event, "error")
		if !pyv.Truthy(message) {
			message = "The GOOSE engine stopped"
		}
		emit("goose_listener_failed", pyv.O("message", message))
	}
}

// Snapshot is {"running", adapter_guid, uplink, share, message, "conflicts": [...]}.
func (e *GooseEngine) Snapshot() *pyjson.Object {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := pyv.O("running", e.process != nil)
	pyv.Update(result, e.link)
	conflicts := []any{}
	for _, id := range e.conflicts.Keys() {
		conflicts = append(conflicts, pyv.Shallow(pyv.GetObj(e.conflicts, id)))
	}
	result.Set("conflicts", conflicts)
	return result
}

// Stop closes the engine and forgets sources, conflicts and the link state.
func (e *GooseEngine) Stop() {
	e.calls.Lock()
	e.mu.Lock()
	process := e.process
	e.process, e.port = nil, nil
	e.sources, e.conflicts = pyjson.NewObject(), pyjson.NewObject()
	e.link = closedLink()
	e.mu.Unlock()
	e.calls.Unlock()
	if process != nil {
		process.Close()
	}
}
