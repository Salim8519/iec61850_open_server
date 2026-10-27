package gateway

import (
	"regexp"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
)

// MasterProtocols are the SCADA master protocols in their stored order.
var MasterProtocols = []string{"iec104", "dnp3", "modbus"}

const maxLabels = 20000

// host is master_rules.HOST; Python's "$" also matches before a final "\n".
var host = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9.\-]{0,251}[A-Za-z0-9])?\n?$`)

// EmptyMaster is empty_master(): the SCADA master block of a new gateway section.
func EmptyMaster() *pyjson.Object {
	settings, unticked, labels := pyjson.NewObject(), pyjson.NewObject(), pyjson.NewObject()
	for _, name := range MasterProtocols {
		settings.Set(name, defaults(masterSettings[name]))
	}
	for _, name := range MasterProtocols {
		unticked.Set(name, []any{})
		labels.Set(name, pyjson.NewObject())
	}
	return pyjson.NewObject().Set("protocol", "iec104").Set("host", "").Set("auto_connect", false).
		Set("settings", settings).Set("modbus_ranges", []any{}).Set("routes", []any{}).
		Set("unticked", unticked).Set("labels", labels)
}

// ProtocolArg is master_rules.protocol_arg.
func ProtocolArg(value any) (any, error) {
	if !pyval.InTuple(value, MasterProtocols...) {
		return nil, apperr.ValueError("Choose IEC 104 (iec104), DNP3 (dnp3) or Modbus TCP (modbus)")
	}
	return value, nil
}

// ValidateHost is master_rules.validate_host: a device address or host name.
func ValidateHost(value any) (string, error) {
	text := ""
	if s, ok := value.(string); ok {
		text = pyrules.Strip(s)
	}
	if !host.MatchString(text) {
		return "", apperr.ValueError("Type the device address, for example 192.168.1.20 or rtu1.station.local")
	}
	return text, nil
}

func masterFlagText(key string) string { return settingLabel(key) + " is on or off" }

// NormalizeMaster is normalize_master: the stored master block with defaults filled in.
func NormalizeMaster(raw any) (*pyjson.Object, error) {
	result := EmptyMaster()
	if raw == nil {
		return result, nil
	}
	stored, isDict := raw.(*pyjson.Object)
	if !isDict {
		return nil, apperr.ValueError("The master section must be an object")
	}
	protocol, err := ProtocolArg(pyval.Lookup(stored, "protocol", "iec104"))
	if err != nil {
		return nil, err
	}
	result.Set("protocol", protocol)
	hostText := ""
	if pyval.Truthy(pyval.Lookup(stored, "host", nil)) {
		if hostText, err = ValidateHost(pyval.Lookup(stored, "host", nil)); err != nil {
			return nil, err
		}
	}
	result.Set("host", hostText)
	result.Set("auto_connect", pyval.Lookup(stored, "auto_connect", nil) == true)
	settings, _ := result.Get("settings")
	unticked, _ := result.Get("unticked")
	labels, _ := result.Get("labels")
	for _, name := range MasterProtocols {
		given, err := pyval.Get(pyval.Or(pyval.Lookup(stored, "settings", nil), pyjson.NewObject()), name, nil)
		if err != nil {
			return nil, err
		}
		checked, err := validateSettings(name, masterSettings[name], given, masterFlagText)
		if err != nil {
			return nil, err
		}
		settings.(*pyjson.Object).Set(name, checked)
		keys, err := untickedKeys(stored, name)
		if err != nil {
			return nil, err
		}
		unticked.(*pyjson.Object).Set(name, keys)
		names, err := pyval.Get(pyval.Or(pyval.Lookup(stored, "labels", nil), pyjson.NewObject()), name, nil)
		if err != nil {
			return nil, err
		}
		labels.(*pyjson.Object).Set(name, cleanLabels(names))
	}
	ranges, err := ValidateRanges(pyval.Lookup(stored, "modbus_ranges", []any{}))
	if err != nil {
		return nil, err
	}
	routes, err := ValidateRoutes(pyval.Lookup(stored, "routes", []any{}))
	if err != nil {
		return nil, err
	}
	result.Set("modbus_ranges", ranges).Set("routes", routes)
	if err := keepPointList(stored, result); err != nil { // Go-only, kept only when stored (ADR-0110)
		return nil, err
	}
	if err := keepAddressList(stored, result, "own_addresses"); err != nil { // ADR-0125
		return nil, err
	}
	if err := keepAddresses(stored, result, "local"); err != nil { // Go-only Network (ADR-0123)
		return nil, err
	}
	return result, nil
}

// untickedKeys is [key for key in unticked if isinstance(key, str)][:20000].
func untickedKeys(stored *pyjson.Object, name string) ([]any, error) {
	value, err := pyval.Get(pyval.Or(pyval.Lookup(stored, "unticked", nil), pyjson.NewObject()), name, []any{})
	if err != nil {
		return nil, err
	}
	items, err := pyval.Iter(value)
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

// cleanLabels is master_rules._labels: short keys with non-blank text, cut to 80 characters.
func cleanLabels(raw any) *pyjson.Object {
	result := pyjson.NewObject()
	table, isDict := raw.(*pyjson.Object)
	if !isDict {
		return result
	}
	for index, key := range table.Keys() {
		if index >= maxLabels {
			break
		}
		value, _ := table.Get(key)
		text, isText := value.(string)
		if pyval.Len(key) <= 40 && isText && pyrules.Strip(text) != "" {
			result.Set(key, pyval.Prefix(text, 80))
		}
	}
	return result
}
