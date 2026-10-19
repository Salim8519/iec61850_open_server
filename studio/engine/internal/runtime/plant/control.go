// Package plant holds the pure equipment rules of the single-line diagram: commands and
// simulated feedback (domain/equipment_control.py), display states (equipment_state.py),
// energisation and switching hints, and test buttons. Node data are the project's ordered
// objects; nothing here reads or writes an IED.
package plant

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Switches are the equipment kinds with an open/closed position.
var Switches = map[string]bool{"CB": true, "DS": true, "ES": true}

// Binding is an exact (IED, reference) a node maps to.
type Binding struct{ IedID, Ref string }

// BindingOf reads {ied_id, ref} from a mapping entry.
func BindingOf(object *pyjson.Object) Binding {
	return Binding{pyval.Str(object, "ied_id"), pyval.Str(object, "ref")}
}

// Target is one feedback write: a binding and the value to write.
type Target struct {
	Binding Binding
	Value   any
}

// ExpectedCDC is expected_cdc.
func ExpectedCDC(kind string) (string, error) {
	if Switches[kind] {
		return "DPC", nil
	}
	if kind == "TRIP_RELAY" || kind == "PUSH_BUTTON" {
		return "SPC", nil
	}
	return "", apperr.ValueError("This equipment type is not controllable")
}

func getOr(data *pyjson.Object, key string, fallback any) any {
	if value, ok := data.Get(key); ok {
		return value
	}
	return fallback
}

// SemanticCommand is semantic_command: (action, wire value). action "" means None.
func SemanticCommand(data *pyjson.Object, action string, value any) (string, bool, error) {
	kind := pyval.Str(data, "kind")
	if action != "" && value != nil {
		return "", false, apperr.ValueError("Send an equipment action or a raw value, not both")
	}
	operated := getOr(data, "trip_command_polarity", "operate_false") == "operate_true"
	if action == "" {
		flag, ok := value.(bool)
		if !ok {
			return "", false, apperr.ValueError("Control value must be boolean")
		}
		switch {
		case Switches[kind]:
			if flag {
				return "close", flag, nil
			}
			return "open", flag, nil
		case kind == "TRIP_RELAY":
			if flag == operated {
				return "operate", flag, nil
			}
			return "reset", flag, nil
		case kind == "PUSH_BUTTON":
			return "press", flag, nil
		}
		return "", false, apperr.ValueError("This equipment type is not controllable")
	}
	switch {
	case Switches[kind] && (action == "open" || action == "close"):
		return action, action == "close", nil
	case kind == "TRIP_RELAY" && (action == "operate" || action == "reset"):
		if action == "operate" {
			return action, operated, nil
		}
		return action, !operated, nil
	case kind == "PUSH_BUTTON" && pyval.Get(data, "push_mode") == "spc_momentary" && action == "press":
		return action, pyval.Truthy(getOr(data, "push_value", true)), nil
	case kind == "PUSH_BUTTON" && pyval.Get(data, "push_mode") == "self_resetting" && action == "press":
		return action, !pyval.Truthy(getOr(data, "push_rest_value", false)), nil
	}
	return "", false, apperr.ValueError("Action is incompatible with this equipment")
}

// FeedbackTargets is feedback_targets: the state points a command writes back.
func FeedbackTargets(data, mapping *pyjson.Object, action string) ([]Target, error) {
	if mapping == nil || mapping.Len() == 0 {
		return nil, nil
	}
	kind := pyval.Str(data, "kind")
	mode := pyval.Str(mapping, "mode")
	if kind == "TRIP_RELAY" {
		if mode != "direct" {
			return nil, apperr.ValueError("Trip relay feedback requires one Boolean state point")
		}
		if action != "operate" && action != "reset" {
			return nil, apperr.ValueError("Trip relay feedback requires an Operate or Reset action")
		}
		operated := getOr(data, "trip_state_polarity", "operated_true") == "operated_true"
		value := operated
		if action != "operate" {
			value = !operated
		}
		return []Target{{BindingOf(pyval.Map(mapping, "source")), value}}, nil
	}
	if !Switches[kind] {
		return nil, apperr.ValueError("This equipment type is not controllable")
	}
	if action != "open" && action != "close" {
		return nil, apperr.ValueError("Switch feedback requires an Open or Close action")
	}
	closed := action == "close"
	switch mode {
	case "direct":
		position := int64(1)
		if closed {
			position = 2
		}
		return []Target{{BindingOf(pyval.Map(mapping, "source")), position}}, nil
	case "single_to_dbpos":
		return []Target{{BindingOf(pyval.Map(mapping, "source")), closed}}, nil
	case "contacts_to_dbpos":
		return []Target{{BindingOf(pyval.Map(mapping, "open")), !closed},
			{BindingOf(pyval.Map(mapping, "closed")), closed}}, nil
	}
	return nil, apperr.ValueError("Unsupported equipment state mapping")
}
