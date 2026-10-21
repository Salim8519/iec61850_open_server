package gateway

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

var deviceKinds = map[string]bool{"single": true, "double": true, "binary_output": true, "coil": true}

// ValidateRoutes is validate_routes(routes): the user's command routes
// [{upstream, upstream_kind, kind, address, select}].
func ValidateRoutes(routes any) ([]any, error) {
	list, ok := routes.([]any)
	if !ok || len(list) > 500 {
		return nil, valueError("Send at most 500 command routes as a list")
	}
	clean := []any{}
	for index, value := range list {
		label := "Route " + itoa(index+1) + ": "
		raw, isDict := value.(*pyjson.Object)
		known := false
		if isDict {
			var err error
			if _, known, err = lookupName(deviceKinds, get(raw, "kind")); err != nil {
				return nil, err
			}
		}
		if !known {
			return nil, valueError(label + "the device command is single, double, binary_output or coil")
		}
		if !contains(CommandKinds, get(raw, "upstream_kind")) {
			return nil, valueError(label + "the upstream command is a single or double command")
		}
		for _, key := range []string{"upstream", "address"} {
			if !pytext.IntIn(get(raw, key), 0, 16777215) {
				return nil, valueError(label + key + " must be a whole number")
			}
		}
		clean = append(clean, pyjson.NewObject().Set("upstream", get(raw, "upstream")).
			Set("upstream_kind", get(raw, "upstream_kind")).Set("kind", get(raw, "kind")).
			Set("address", get(raw, "address")).Set("select", getOr(raw, "select", true) == true))
	}
	return clean, nil
}

// labelsOf is _labels(raw): str labels of at most 80 characters for keys of at most 40.
func labelsOf(raw any) *pyjson.Object {
	result := pyjson.NewObject()
	object, ok := raw.(*pyjson.Object)
	if !ok {
		return result
	}
	keys := object.Keys()
	if len(keys) > maxLabels {
		keys = keys[:maxLabels]
	}
	for _, key := range keys {
		value, _ := object.Get(key)
		text, isText := value.(string)
		if pytext.Len(key) <= 40 && isText && pytext.Strip(text) != "" {
			result.Set(key, pytext.Head(text, 80))
		}
	}
	return result
}

// untickedOf is [key for key in unticked if isinstance(key, str)][:MAX_LABELS].
func untickedOf(value any) ([]any, error) {
	items, err := iterate(value)
	if err != nil {
		return nil, err
	}
	keys := []any{}
	for _, item := range items {
		if _, ok := item.(string); ok && len(keys) < maxLabels {
			keys = append(keys, item)
		}
	}
	return keys, nil
}

// NormalizeMaster is normalize_master(raw): the stored master block as a checked structure
// (missing parts get the defaults).
func NormalizeMaster(raw any) (*pyjson.Object, error) {
	result := EmptyMaster()
	if raw == nil {
		return result, nil
	}
	section, ok := raw.(*pyjson.Object)
	if !ok {
		return nil, valueError("The master section must be an object")
	}
	protocol, err := ProtocolArg(getOr(section, "protocol", "iec104"))
	if err != nil {
		return nil, err
	}
	result.Set("protocol", protocol)
	host := ""
	if pytext.Truthy(get(section, "host")) {
		if host, err = ValidateHost(get(section, "host")); err != nil {
			return nil, err
		}
	}
	result.Set("host", host).Set("auto_connect", get(section, "auto_connect") == true)
	settings := get(result, "settings").(*pyjson.Object)
	unticked := get(result, "unticked").(*pyjson.Object)
	labels := get(result, "labels").(*pyjson.Object)
	for _, name := range MasterProtocols {
		values, err := orGet(get(section, "settings"), name, nil)
		if err != nil {
			return nil, err
		}
		checked, err := ValidateMasterSettings(name, values)
		if err != nil {
			return nil, err
		}
		settings.Set(name, checked)
		keys, err := orGet(get(section, "unticked"), name, []any{})
		if err == nil {
			keys, err = untickedOf(keys)
		}
		if err != nil {
			return nil, err
		}
		unticked.Set(name, keys)
		names, err := orGet(get(section, "labels"), name, nil)
		if err != nil {
			return nil, err
		}
		labels.Set(name, labelsOf(names))
	}
	ranges, err := ValidateRanges(getOr(section, "modbus_ranges", []any{}))
	if err != nil {
		return nil, err
	}
	routes, err := ValidateRoutes(getOr(section, "routes", []any{}))
	if err != nil {
		return nil, err
	}
	return result.Set("modbus_ranges", ranges).Set("routes", routes), nil
}
