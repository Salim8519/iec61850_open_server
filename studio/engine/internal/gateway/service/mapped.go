package service

import (
	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/masterpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Mapped is what the station gateway's own mapping says the gateway point at (protocol,
// address, kind) carries now: the value of the IED signal behind it (or the made-up value of
// an own point), whether that is current, and the gateway point's scale. It is the gateway's
// view for the master's optional "Check against IED" test aid (ADR-0110); the master itself
// never reads IEDs. The gateway worker does not need to run: the value comes from the runtime.
func (m *Manager) Mapped(protocol string, address int64, kind string) masterpoints.Mapped {
	lists, _ := get(m.Config(), "points").(*pyjson.Object)
	points, _ := get(lists, protocol).([]any)
	for _, raw := range points {
		point, ok := raw.(*pyjson.Object)
		at, isInt := get(point, "address").(int64)
		if !ok || !isInt || at != address || text(point, "kind") != kind {
			continue
		}
		scale := 1.0
		if value, found := point.Get("scale"); found {
			if number, isNumber := pytext.Float(value); isNumber {
				scale = number
			}
		}
		if gateway.IsOwn(point) {
			sample := m.own.current(text(point, "id"), get(point, "value"))
			return masterpoints.Mapped{Found: true, Online: true, Value: sample.Value, Scale: scale,
				Source: "made-up value of the gateway"}
		}
		iedID, ref := text(point, "ied"), text(point, "ref")
		ied := m.IED(iedID)
		sample := SampleOf(text(point, "id"), iedID, ref, ied, m.runtime.RunState(), m.runtime)
		name := text(ied, "name")
		if name == "" {
			name = iedID
		}
		return masterpoints.Mapped{Found: true, Online: sample.SourceOK, Value: sample.Value, Scale: scale,
			Source: name + " " + ref}
	}
	return masterpoints.Mapped{}
}
