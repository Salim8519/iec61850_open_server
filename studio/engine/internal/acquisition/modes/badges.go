package modes

import (
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// ModeBadges is mode_badges: {ieds: {id: summary}, equipment: {node id: text}} for
// anything not in normal service. Simulated IEDs show the mode Studio set; real IEDs (only
// while running) the mode read from the device. sessions maps an IED id to its live
// session status; a missing key or nil value is Python's None.
func ModeBadges(ieds []*pyjson.Object, sessions map[string]*pyjson.Object, diagram *pyjson.Object,
	running bool) *pyjson.Object {
	summaries := pyjson.NewObject()
	for _, ied := range ieds {
		id := pyv.GetStr(ied, "id")
		role := pyv.Get(ied, "role")
		if pyv.Equal(role, "simulated") {
			summaries.Set(id, simulatedSummary(ied))
		} else if pyv.Equal(role, "real") && running {
			health := pyv.Get(sessions[id], "health")
			if pyv.Truthy(health) {
				mode := pyv.GetObj(pyv.Obj(health), "mode")
				summaries.Set(id, ModeSummary(ied, pyv.GetObj(mode, "values")))
			}
		}
	}
	equipment := pyjson.NewObject()
	for _, node := range pyv.Objects(pyv.GetList(diagram, "nodes")) {
		iedID, domain, bound := boundIED(pyv.GetObj(node, "data"))
		var summary *pyjson.Object
		var name any
		if bound {
			if text, isText := iedID.(string); isText {
				summary = pyv.GetObj(summaries, text)
			}
			name = pyv.Get(pyv.GetObj(summary, "devices"), domain)
		}
		key := pyv.PyStr(pyv.Get(node, "id"))
		if pyv.Truthy(name) && !pyv.Equal(name, "on") && !pyv.Equal(name, "not available") {
			equipment.Set(key, name)
		} else if filled(summary) && pyv.Truthy(pyv.Get(summary, "simulation")) {
			equipment.Set(key, "simulation")
		}
	}
	return pyv.O("ieds", summaries, "equipment", equipment)
}

// simulatedSummary is _simulated_summary: the saved test mode of a simulated IED. A
// stored mode outside MODE_NAMES (Python raises KeyError; validation prevents it) shows
// as "not available" and does not count for the worst mode.
func simulatedSummary(ied *pyjson.Object) *pyjson.Object {
	settings := pyv.OrObj(ied, "test_mode")
	devices, _ := ModePoints(ied)
	wanted := pyv.Obj(pyv.GetOr(settings, "devices", pyjson.NewObject()))
	names := pyjson.NewObject()
	var worst any
	for _, device := range devices {
		mode, known := modeKey(pyv.GetOr(wanted, device.Domain, int64(1)))
		if !known {
			names.Set(device.Domain, "not available")
			continue
		}
		name := ModeNames()[mode-1]
		names.Set(device.Domain, name)
		if worst == nil || modeRank(name) > modeRank(worst.(string)) {
			worst = name
		}
	}
	return pyv.O("devices", names, "simulation", pyv.Truthy(pyv.Get(settings, "simulation")), "worst", worst)
}

// boundIED is _bound_ied: the IED id and logical device a diagram node is bound to, from
// its control binding or else its state mapping (source, else open).
func boundIED(data *pyjson.Object) (any, string, bool) {
	mapping := pyv.OrObj(data, "state_mapping")
	source := pyv.Get(mapping, "source")
	if !pyv.Truthy(source) {
		source = pyv.Get(mapping, "open")
	}
	for _, candidate := range []any{pyv.Get(data, "control_binding"), source} {
		item := pyv.Obj(candidate)
		if !filled(item) || !pyv.Truthy(pyv.Get(item, "ied_id")) {
			continue
		}
		ref, isText := pyv.GetOr(item, "ref", "").(string)
		if !isText || !strings.Contains(ref, "/") {
			continue
		}
		domain, _, _ := strings.Cut(ref, "/")
		return pyv.Get(item, "ied_id"), domain, true
	}
	return nil, "", false
}

// filled is Python truthiness of an object that may be missing (nil): non-empty.
func filled(object *pyjson.Object) bool {
	return object != nil && object.Len() > 0
}
