package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// bindingKeys is equipment_state.binding_keys: the (IED, reference) pairs of a mapping.
func bindingKeys(mapping any) [][2]string {
	if !pyval.Truthy(mapping) {
		return nil
	}
	m := object(mapping)
	names := []string{"source"}
	if text(m, "mode") == "contacts_to_dbpos" {
		names = []string{"open", "closed"}
	}
	keys := [][2]string{}
	for _, name := range names {
		binding := object(pyval.Lookup(m, name, nil))
		keys = append(keys, [2]string{text(binding, "ied_id"), text(binding, "ref")})
	}
	return keys
}

func usesIED(data *pyjson.Object, ids map[string]bool) bool {
	for _, key := range bindingKeys(pyval.Lookup(data, "state_mapping", nil)) {
		if ids[key[0]] {
			return true
		}
	}
	for _, entry := range items(pyval.Or(pyval.Lookup(data, "relay_statuses", nil), []any{})) {
		if ids[text(object(pyval.Lookup(object(entry), "source", nil)), "ied_id")] {
			return true
		}
	}
	if measurementUsesIED(data, ids) || pointPanelUsesIED(data, ids) || tapChangerUsesIED(data, ids) ||
		gooseUsesIED(data, ids) {
		return true
	}
	control := pyval.Lookup(data, "control_binding", nil)
	return pyval.Truthy(control) && ids[text(object(control), "ied_id")]
}

func diagramNodes(project *pyjson.Object) []any {
	return items(pyval.Lookup(object(pyval.Lookup(project, "diagram", nil)), "nodes", nil))
}

// Impacts is removal.impacts: what removing the IEDs would change, in Python's key order.
func Impacts(project *pyjson.Object, ids map[string]bool) *pyjson.Object {
	bindings, steps, removed := 0, 0, 0
	for _, node := range diagramNodes(project) {
		if usesIED(object(pyval.Lookup(object(node), "data", nil)), ids) {
			bindings++
		}
	}
	for _, item := range items(pyval.Lookup(project, "scenarios", nil)) {
		all := true
		for _, step := range items(pyval.Lookup(object(item), "steps", nil)) {
			if ids[text(object(step), "ied_id")] {
				steps++
			} else {
				all = false
			}
		}
		if all {
			removed++
		}
	}
	count := func(key string) int64 {
		total := int64(0)
		for _, entry := range items(pyval.Lookup(project, key, []any{})) {
			if ids[text(object(entry), "ied_id")] {
				total++
			}
		}
		return total
	}
	return pyjson.NewObject().Set("bindings", int64(bindings)).Set("sources", count("sources")).
		Set("test_controls", count("test_panel")).Set("scenario_steps", int64(steps)).
		Set("scenarios_removed", int64(removed))
}

// WithoutIEDs is removal.without_ieds: a deep copy without the IEDs and everything that
// referred to them; diagram equipment stays, only its bindings are cleared. The Go-only IED
// component of a removed IED is removed with it (ADR-0112).
func WithoutIEDs(project *pyjson.Object, ids map[string]bool) *pyjson.Object {
	result := pyval.CopyObject(project)
	keep := func(key string) []any {
		kept := []any{}
		for _, entry := range items(pyval.Lookup(result, key, []any{})) {
			if !ids[text(object(entry), "ied_id")] {
				kept = append(kept, entry)
			}
		}
		return kept
	}
	ieds := []any{}
	for _, ied := range items(pyval.Lookup(result, "ieds", nil)) {
		if !ids[text(object(ied), "id")] {
			ieds = append(ieds, ied)
		}
	}
	result.Set("ieds", ieds).Set("sources", keep("sources")).Set("test_panel", keep("test_panel"))
	for _, node := range diagramNodes(result) {
		data := object(pyval.Lookup(object(node), "data", nil))
		for _, key := range bindingKeys(pyval.Lookup(data, "state_mapping", nil)) {
			if ids[key[0]] {
				data.Set("state_mapping", nil)
				break
			}
		}
		if statuses := pyval.Lookup(data, "relay_statuses", nil); pyval.Truthy(statuses) {
			kept := []any{}
			for _, entry := range items(statuses) {
				if !ids[text(object(pyval.Lookup(object(entry), "source", nil)), "ied_id")] {
					kept = append(kept, entry)
				}
			}
			data.Set("relay_statuses", kept)
		}
		dropMeasurementsOf(data, ids)
		dropPointPanelOf(data, ids)
		dropTapChangerOf(data, ids)
		dropGooseLinkOf(data, ids)
		if control := pyval.Lookup(data, "control_binding", nil); pyval.Truthy(control) && ids[text(object(control), "ied_id")] {
			data.Set("control_binding", nil)
		}
	}
	if diagram := object(pyval.Lookup(result, "diagram", nil)); diagram != nil {
		withoutIEDComponents(diagram, ids) // ADR-0112: an IED's component goes with it
	}
	scenarios := []any{}
	for _, item := range items(pyval.Lookup(result, "scenarios", nil)) {
		scenario := object(item)
		steps := []any{}
		for _, step := range items(pyval.Lookup(scenario, "steps", nil)) {
			if !ids[text(object(step), "ied_id")] {
				steps = append(steps, step)
			}
		}
		scenario.Set("steps", steps)
		if len(steps) > 0 {
			scenarios = append(scenarios, scenario)
		}
	}
	return result.Set("scenarios", scenarios)
}

// FindIED is ProjectService.find_ied: the IED with this id or a not_found StudioError.
func FindIED(project *pyjson.Object, key string) (*pyjson.Object, error) {
	for _, item := range items(pyval.Lookup(project, "ieds", nil)) {
		if ied := object(item); text(ied, "id") == key {
			return ied, nil
		}
	}
	return nil, apperr.Coded("Unknown IED", "not_found")
}

// RemovalPreview is ProjectService.removal_preview: {id, name, **impact}.
func RemovalPreview(project *pyjson.Object, key string) (*pyjson.Object, error) {
	ied, err := FindIED(project, key)
	if err != nil {
		return nil, err
	}
	result := pyjson.NewObject().Set("id", key).Set("name", pyval.Lookup(ied, "name", nil))
	return appendAll(result, Impacts(project, map[string]bool{key: true})), nil
}

func appendAll(result, extra *pyjson.Object) *pyjson.Object {
	for _, key := range extra.Keys() {
		value, _ := extra.Get(key)
		result.Set(key, value)
	}
	return result
}
