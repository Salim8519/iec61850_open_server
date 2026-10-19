package netset

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// isModeName is `mode in MODE_NAMES` ({1..5: name}): ints and equal floats are members,
// lists and dicts cannot be hashed.
func isModeName(mode any) (bool, error) {
	if err := pyval.CheckHashable(mode); err != nil {
		return false, err
	}
	for value := 1; value <= 5; value++ {
		if pyval.Equal(mode, int64(value)) {
			return true, nil
		}
	}
	return false, nil
}

// ValidateTestMode is validate_test_mode: stored per-IED test settings
// {devices: {domain: 1..5}, simulation: bool}.
func ValidateTestMode(value any) (*pyjson.Object, error) {
	value = pyval.Or(value, pyjson.NewObject())
	devices, err := pyval.Get(value, "devices", nil)
	if err != nil {
		return nil, err
	}
	devices = pyval.Or(devices, pyjson.NewObject())
	table, isDict := devices.(*pyjson.Object)
	if !isDict || table.Len() > 100 {
		return nil, apperr.ValueError("Test mode needs one mode per logical device")
	}
	clean := pyjson.NewObject()
	for _, domain := range table.Keys() {
		mode, _ := table.Get(domain)
		_, isBool := mode.(bool)
		member := false
		if !isBool {
			if member, err = isModeName(mode); err != nil {
				return nil, err
			}
		}
		if isBool || !member {
			return nil, apperr.ValueError("Each logical device mode must be 1 (on) to 5 (off)")
		}
		clean.Set(domain, mode)
	}
	simulation, _ := pyval.Get(value, "simulation", false)
	if _, ok := simulation.(bool); !ok {
		return nil, apperr.ValueError("Simulation must be on or off")
	}
	return pyjson.NewObject().Set("devices", clean).Set("simulation", simulation), nil
}
