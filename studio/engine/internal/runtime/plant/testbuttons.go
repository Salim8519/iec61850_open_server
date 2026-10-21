package plant

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Test buttons on simulated IEDs (domain/equipment_test.py): writes as if the plant changed.
var (
	dbposOf  = map[string]int64{"moving": 0, "open": 1, "closed": 2, "bad": 3}
	contacts = map[string][2]bool{"open": {true, false}, "closed": {false, true}, "moving": {false, false}, "bad": {true, true}}
)

// TestWrite is one (binding, value, quality) write; Value or Quality nil mean None.
type TestWrite struct {
	Binding Binding
	Value   any
	Quality any
}

// ButtonActions is button_actions.
func ButtonActions(data *pyjson.Object) []string {
	kind, mapping := pyval.Str(data, "kind"), pyval.Map(data, "state_mapping")
	if kind == "RELAY" {
		if len(pyval.List(data, "relay_statuses")) > 0 {
			return []string{"on", "off"}
		}
		return nil
	}
	if mapping == nil || mapping.Len() == 0 {
		return nil
	}
	switch {
	case Switches[kind]:
		if pyval.Str(mapping, "mode") == "single_to_dbpos" {
			return []string{"open", "closed"}
		}
		return []string{"open", "closed", "moving", "bad"}
	case kind == "TRIP_RELAY":
		return []string{"trip", "reset"}
	case kind == "CT" || kind == "VT":
		return []string{"set", "bad_quality"}
	case kind == "SOURCE" || kind == "INFEED":
		return []string{"live", "dead"}
	}
	return nil
}

// ButtonWrites is button_writes; kindOf gives the SCL kind of a bound point ("" = unknown).
func ButtonWrites(data *pyjson.Object, action any, value any, row any, kindOf func(Binding) string) ([]TestWrite, error) {
	name, _ := action.(string)
	allowed := false
	for _, candidate := range ButtonActions(data) {
		allowed = allowed || (name == candidate && pyval.Equal(action, candidate))
	}
	if !allowed {
		return nil, apperr.ValueError("This test action does not fit this equipment")
	}
	kind, mapping := pyval.Str(data, "kind"), pyval.Map(data, "state_mapping")
	if kind == "RELAY" {
		for _, item := range pyval.List(data, "relay_statuses") {
			entry := item.(*pyjson.Object)
			if pyval.Equal(pyval.Get(entry, "id"), row) {
				return []TestWrite{{BindingOf(pyval.Map(entry, "source")), name == "on", nil}}, nil
			}
		}
		return nil, apperr.ValueError("Choose a relay panel row")
	}
	if Switches[kind] {
		switch pyval.Str(mapping, "mode") {
		case "direct":
			return []TestWrite{{BindingOf(pyval.Map(mapping, "source")), dbposOf[name], nil}}, nil
		case "single_to_dbpos":
			return []TestWrite{{BindingOf(pyval.Map(mapping, "source")), name == "closed", nil}}, nil
		}
		pair := contacts[name]
		return []TestWrite{{BindingOf(pyval.Map(mapping, "open")), pair[0], nil},
			{BindingOf(pyval.Map(mapping, "closed")), pair[1], nil}}, nil
	}
	source := BindingOf(pyval.Map(mapping, "source"))
	switch {
	case kind == "TRIP_RELAY":
		operated := getOr(data, "trip_state_polarity", "operated_true") == "operated_true"
		if name == "trip" {
			return []TestWrite{{source, operated, nil}}, nil
		}
		return []TestWrite{{source, !operated, nil}}, nil
	case kind == "CT" || kind == "VT":
		if name == "bad_quality" {
			return []TestWrite{{source, nil, int64(2)}}, nil
		}
		if !pyval.IsNumber(value) {
			return nil, apperr.ValueError("Enter a number")
		}
		return []TestWrite{{source, value, int64(0)}}, nil
	}
	live := name == "live"
	if kindOf(source) == "Dbpos" {
		position := int64(1)
		if live {
			position = 2
		}
		return []TestWrite{{source, position, nil}}, nil
	}
	return []TestWrite{{source, live, nil}}, nil
}
