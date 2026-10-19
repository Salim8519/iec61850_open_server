package plant

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// BindingKeys is equipment_state.binding_keys.
func BindingKeys(mapping *pyjson.Object) []Binding {
	if mapping == nil || mapping.Len() == 0 {
		return nil
	}
	if pyval.Str(mapping, "mode") == "contacts_to_dbpos" {
		return []Binding{BindingOf(pyval.Map(mapping, "open")), BindingOf(pyval.Map(mapping, "closed"))}
	}
	return []Binding{BindingOf(pyval.Map(mapping, "source"))}
}

// EvaluateState is evaluate_state: a raw direct value or a normalised Dbpos integer (nil = None).
func EvaluateState(mapping *pyjson.Object, values func(Binding) any) any {
	keys := BindingKeys(mapping)
	switch pyval.Str(mapping, "mode") {
	case "direct":
		return values(keys[0])
	case "single_to_dbpos":
		switch values(keys[0]) {
		case true:
			return int64(2)
		case false:
			return int64(1)
		}
		return nil
	}
	openValue, openOK := values(keys[0]).(bool)
	closedValue, closedOK := values(keys[1]).(bool)
	if !openOK || !closedOK {
		return nil
	}
	switch {
	case openValue && !closedValue:
		return int64(1)
	case !openValue && closedValue:
		return int64(2)
	case openValue && closedValue:
		return int64(3)
	}
	return int64(0)
}
