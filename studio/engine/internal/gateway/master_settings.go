package gateway

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Roles are the gateway roles: station gateway, SCADA master, protocol converter (ADR-0061).
var Roles = []string{"gateway", "master", "converter"}

// MasterProtocols are the SCADA master protocols in master_rules order.
var MasterProtocols = []string{"iec104", "dnp3", "modbus"}

const maxLabels = 20000

var masterSettings = map[string]settingsRules{
	"iec104": masterRules(setting{"port", int64(2404)}, setting{"common_address", int64(1)},
		setting{"originator_address", int64(0)}, setting{"t0", int64(10)}, setting{"t1", int64(15)},
		setting{"t2", int64(10)}, setting{"t3", int64(20)}, setting{"k", int64(12)}, setting{"w", int64(8)},
		setting{"time_tagged_commands", false}, setting{"gi_minutes", int64(15)}),
	"dnp3": masterRules(setting{"port", int64(20000)}, setting{"outstation_address", int64(10)},
		setting{"master_address", int64(1)}, setting{"integrity_minutes", int64(15)},
		setting{"event_seconds", int64(5)}, setting{"unsolicited", true}),
	"modbus": masterRules(setting{"port", int64(502)}, setting{"unit_id", int64(1)}, setting{"poll_ms", int64(1000)}),
}

var masterRanges = map[string]limits{"port": {1, 65535}, "common_address": {1, 65534},
	"originator_address": {0, 255}, "t0": {1, 255}, "t1": {1, 255}, "t2": {1, 255}, "t3": {1, 3600},
	"k": {1, 32767}, "w": {1, 32767}, "gi_minutes": {0, 1440}, "outstation_address": {0, 65519},
	"master_address": {0, 65519}, "integrity_minutes": {0, 1440}, "event_seconds": {0, 3600},
	"unit_id": {0, 255}, "poll_ms": {200, 60000}}

func masterRules(defaults ...setting) settingsRules {
	return settingsRules{defaults: defaults, ranges: masterRanges, flagText: func(key string) (string, bool) {
		return settingLabel(key) + " is on or off", key == "time_tagged_commands" || key == "unsolicited"
	}}
}

// EmptyMaster is empty_master().
func EmptyMaster() *pyjson.Object {
	settings, unticked, labels := pyjson.NewObject(), pyjson.NewObject(), pyjson.NewObject()
	for _, name := range MasterProtocols {
		settings.Set(name, masterSettings[name].object())
		unticked.Set(name, []any{})
		labels.Set(name, pyjson.NewObject())
	}
	return pyjson.NewObject().Set("protocol", "iec104").Set("host", "").Set("auto_connect", false).
		Set("settings", settings).Set("modbus_ranges", []any{}).Set("routes", []any{}).
		Set("unticked", unticked).Set("labels", labels)
}

// ProtocolArg is master_rules.protocol_arg(value).
func ProtocolArg(value any) (string, error) {
	if !contains(MasterProtocols, value) {
		return "", valueError("Choose IEC 104 (iec104), DNP3 (dnp3) or Modbus TCP (modbus)")
	}
	return value.(string), nil
}

// ValidateRole is validate_role(value).
func ValidateRole(value any) (string, error) {
	if !contains(Roles, value) {
		return "", valueError("Choose Station gateway (gateway), SCADA master (master) or Protocol converter (converter)")
	}
	return value.(string), nil
}

// ValidateHost is validate_host(value): a host name or IPv4 text, 1 to 253 characters.
func ValidateHost(value any) (string, error) {
	host := ""
	if text, ok := value.(string); ok {
		host = pytext.Strip(text)
	}
	if !hostPattern(host) {
		return "", valueError("Type the device address, for example 192.168.1.20 or rtu1.station.local")
	}
	return host, nil
}

// hostPattern is ^[A-Za-z0-9]([A-Za-z0-9.\-]{0,251}[A-Za-z0-9])?$ on a stripped str ("$"
// before a final newline cannot apply: strip removed it).
func hostPattern(host string) bool {
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for index := 0; index < len(host); index++ {
		c := host[index]
		edge := index == 0 || index == len(host)-1
		alnum := (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
		if !alnum && (edge || (c != '.' && c != '-')) {
			return false
		}
	}
	return true
}

// ValidateMasterSettings is master_rules.validate_settings(protocol, values).
func ValidateMasterSettings(protocol any, values any) (*pyjson.Object, error) {
	name, err := ProtocolArg(protocol)
	if err != nil {
		return nil, err
	}
	return masterSettings[name].check(name, values)
}
