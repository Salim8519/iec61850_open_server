package acquisition

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// WatchedLimit caps the points a real-IED session polls (manager.py:29).
const WatchedLimit = 500

// BindingKeys is domain/equipment_state.binding_keys: the (IED, ref) pairs of a mapping.
func BindingKeys(mapping *pyjson.Object) []PointKey {
	if mapping == nil || mapping.Len() == 0 {
		return nil
	}
	var bindings []*pyjson.Object
	if pyv.GetStr(mapping, "mode") == "contacts_to_dbpos" {
		bindings = []*pyjson.Object{pyv.GetObj(mapping, "open"), pyv.GetObj(mapping, "closed")}
	} else {
		bindings = []*pyjson.Object{pyv.GetObj(mapping, "source")}
	}
	keys := make([]PointKey, 0, len(bindings))
	for _, item := range bindings {
		keys = append(keys, PointKey{IED: pyv.GetStr(item, "ied_id"), Ref: pyv.GetStr(item, "ref")})
	}
	return keys
}

// WatchedPoints lists the signals of a real IED that the canvas or test panel shows, plus
// their quality and time points, as [{ref, fc}] (at most 500).
//
// Python collects the keys in a set, so its order (and which 500 survive a larger set)
// depends on string hashing per process; Go uses the deterministic first-seen order:
// diagram state mappings, relay statuses, then the test panel.
func WatchedPoints(project, ied *pyjson.Object) []any {
	var keys []PointKey
	seen := map[PointKey]bool{}
	add := func(found []PointKey) {
		for _, key := range found {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	nodes := pyv.Objects(pyv.GetList(pyv.GetObj(project, "diagram"), "nodes"))
	for _, node := range nodes {
		add(BindingKeys(pyv.GetObj(pyv.GetObj(node, "data"), "state_mapping")))
	}
	for _, node := range nodes {
		for _, entry := range pyv.Objects(pyv.GetList(pyv.GetObj(node, "data"), "relay_statuses")) {
			add(BindingKeys(pyv.O("mode", "direct", "source", pyv.Get(entry, "source"))))
		}
	}
	for _, entry := range pyv.Objects(pyv.GetList(project, "test_panel")) {
		add([]PointKey{{IED: pyv.GetStr(entry, "ied_id"), Ref: pyv.GetStr(entry, "ref")}})
	}
	byRef := map[string]*pyjson.Object{}
	for _, signal := range pyv.Objects(pyv.GetList(ied, "signals")) {
		byRef[pyv.GetStr(signal, "ref")] = signal
	}
	iedID := pyv.GetStr(ied, "id")
	points := []any{}
	taken := map[string]bool{}
	for _, key := range keys {
		signal := byRef[key.Ref]
		if key.IED != iedID || signal == nil {
			continue
		}
		for _, ref := range []any{key.Ref, pyv.Get(signal, "quality_ref"), pyv.Get(signal, "timestamp_ref")} {
			text, ok := ref.(string)
			if !ok || text == "" || byRef[text] == nil || taken[text] {
				continue
			}
			taken[text] = true
			points = append(points, pyv.O("ref", text, "fc", pyv.Get(byRef[text], "fc")))
		}
	}
	if len(points) > WatchedLimit {
		points = points[:WatchedLimit]
	}
	return points
}
