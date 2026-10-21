package project

import (
	"sort"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
	"studio-engine/internal/scl/value"
)

// ValidateScenarios is domain/scenario.validate_scenarios: timed scenario definitions,
// each step sorted by time (stable). Errors are ValueError, as in Python.
func ValidateScenarios(raw any, ieds []*pyjson.Object) ([]any, error) {
	list, isList := raw.([]any)
	if !isList || len(list) > 100 {
		return nil, apperr.ValueError("Expected at most 100 scenarios")
	}
	signals := signalIndex(ieds)
	result, ids := []any{}, map[string]bool{}
	for _, item := range list {
		scenario, isDict := item.(*pyjson.Object)
		if !isDict {
			return nil, apperr.ValueError("Invalid scenario definition")
		}
		key := pyval.Lookup(scenario, "id", "")
		if !validIdentifier(key) || ids[key.(string)] {
			return nil, apperr.ValueError("Scenario identifiers must be valid and unique")
		}
		ids[key.(string)] = true
		name, isText := pyval.Lookup(scenario, "name", "").(string)
		if length := pyval.Len(pyrules.Strip(name)); !isText || length < 1 || length > 100 {
			return nil, apperr.ValueError("Scenario name must contain 1 to 100 characters")
		}
		enabled := pyval.Lookup(scenario, "enabled", false)
		if _, ok := enabled.(bool); !ok {
			return nil, apperr.ValueError("Scenario enabled must be true or false")
		}
		steps, isList := pyval.Lookup(scenario, "steps", nil).([]any)
		if !isList || len(steps) < 1 || len(steps) > 1000 {
			return nil, apperr.ValueError("Each scenario needs 1 to 1000 steps")
		}
		normalized, err := validateSteps(steps, signals)
		if err != nil {
			return nil, err
		}
		result = append(result, pyjson.NewObject().Set("id", key).Set("name", pyrules.Strip(name)).
			Set("enabled", enabled).Set("steps", normalized))
	}
	return result, nil
}

func validateSteps(steps []any, signals map[signalKey]*pyjson.Object) ([]any, error) {
	normalized, stepIDs := []any{}, map[string]bool{}
	times := []float64{}
	for _, item := range steps {
		step, isDict := item.(*pyjson.Object)
		if !isDict {
			return nil, apperr.ValueError("Invalid scenario step")
		}
		stepID := pyval.Lookup(step, "id", "")
		if !validIdentifier(stepID) || stepIDs[stepID.(string)] {
			return nil, apperr.ValueError("Step identifiers must be valid and unique")
		}
		stepIDs[stepID.(string)] = true
		at := pyval.Lookup(step, "at", nil)
		if !pyval.IsNumber(at) {
			return nil, apperr.ValueError("Step time must be 0 to 86400 seconds")
		}
		if finite, err := pyval.IsFinite(at); err != nil {
			return nil, err
		} else if !finite || !pyval.Between(at, 0, 86400) {
			return nil, apperr.ValueError("Step time must be 0 to 86400 seconds")
		}
		iedID, idIsText := pyval.Lookup(step, "ied_id", nil).(string)
		reference, refIsText := pyval.Lookup(step, "ref", nil).(string)
		if !idIsText || !refIsText {
			return nil, apperr.ValueError("Choose a signal for each scenario step")
		}
		signal := signals[signalKey{iedID, reference}]
		action := pyval.Lookup(step, "action", "set")
		if signal == nil {
			return nil, apperr.ValueError("Unknown scenario signal or action")
		}
		if known, err := pyval.InStrings(action, "set", "assert"); err != nil {
			return nil, err
		} else if !known {
			return nil, apperr.ValueError("Unknown scenario signal or action")
		}
		if action == "set" && (!pyval.Truthy(pyval.Lookup(signal, "writable", nil)) || text(signal, "fc") == "CF") {
			return nil, apperr.ValueError("Scenario cannot write this signal")
		}
		expected, err := value.Validate(signal, pyval.Lookup(step, "value", nil))
		if err != nil {
			return nil, err
		}
		seconds := pyval.AsFloat(at)
		times = append(times, seconds)
		normalized = append(normalized, pyjson.NewObject().Set("id", stepID).Set("at", seconds).
			Set("ied_id", iedID).Set("ref", reference).Set("action", action).Set("value", expected))
	}
	order := make([]int, len(normalized))
	for index := range order {
		order[index] = index
	}
	sort.SliceStable(order, func(a, b int) bool { return times[order[a]] < times[order[b]] })
	sorted := make([]any, len(normalized))
	for index, position := range order {
		sorted[index] = normalized[position]
	}
	return sorted, nil
}
