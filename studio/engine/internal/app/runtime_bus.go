package app

import (
	"studio-engine/internal/events"
	"studio-engine/internal/pyjson"
)

// runtimeBus is the SSE ring as runtime.Bus: Publish returns the envelope
// {"id", "kind", "timestamp", **payload} that the coordinator appends to the run log, as
// event_bus.publish returned it in Python.
type runtimeBus struct {
	bus  *events.Bus
	logf func(string, ...any)
}

func (b runtimeBus) Publish(kind string, payload *pyjson.Object) *pyjson.Object {
	event := publish(b.bus, b.logf, kind, payload)
	if event == nil {
		return nil
	}
	envelope := pyjson.NewObject().Set("id", event.ID).Set("kind", event.Kind).Set("timestamp", event.Timestamp)
	for _, key := range event.Payload.Keys() {
		value, _ := event.Payload.Get(key)
		envelope.Set(key, value) // a payload "id"/"kind"/"timestamp" keeps its place, as in {**payload}
	}
	return envelope
}
