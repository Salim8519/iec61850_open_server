package modes

import (
	"errors"
	"fmt"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// The ValueError texts of validate_test_mode.
const (
	errDevices    = "Test mode needs one mode per logical device"
	errDeviceMode = "Each logical device mode must be 1 (on) to 5 (off)"
	errSimulation = "Simulation must be on or off"
)

// ValidateTestMode is validate_test_mode: the stored per-IED test settings
// {devices: {domain: 1..5}, simulation: bool}. The error text is Python's ValueError text.
//
// Python raises AttributeError for a truthy value that is not an object and TypeError for
// a list or object used as a device mode; Go returns an error with Python's exception text
// for those too (such input never comes from the Studio interface).
func ValidateTestMode(value any) (*pyjson.Object, error) {
	if !pyv.Truthy(value) {
		value = pyjson.NewObject()
	}
	settings, ok := value.(*pyjson.Object)
	if !ok {
		return nil, fmt.Errorf("'%s' object has no attribute 'get'", typeName(value))
	}
	devicesValue := pyv.Get(settings, "devices")
	if !pyv.Truthy(devicesValue) {
		devicesValue = pyjson.NewObject()
	}
	devices, ok := devicesValue.(*pyjson.Object)
	if !ok || devices.Len() > 100 {
		return nil, errors.New(errDevices)
	}
	clean := pyjson.NewObject()
	for _, domain := range devices.Keys() {
		mode, _ := devices.Get(domain)
		if _, isBool := mode.(bool); isBool {
			return nil, errors.New(errDeviceMode)
		}
		switch mode.(type) {
		case []any, *pyjson.Object:
			return nil, fmt.Errorf("unhashable type: '%s'", typeName(mode))
		}
		if _, known := modeKey(mode); !known {
			return nil, errors.New(errDeviceMode)
		}
		clean.Set(domain, mode)
	}
	simulation := pyv.GetOr(settings, "simulation", false)
	if _, isBool := simulation.(bool); !isBool {
		return nil, errors.New(errSimulation)
	}
	return pyv.O("devices", clean, "simulation", simulation), nil
}

// typeName is Python's type(value).__name__ for pyjson values.
func typeName(value any) string {
	switch value.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case float64:
		return "float"
	case string:
		return "str"
	case []any:
		return "list"
	case *pyjson.Object:
		return "dict"
	}
	if pyv.IsInt(value) {
		return "int"
	}
	return fmt.Sprintf("%T", value)
}
