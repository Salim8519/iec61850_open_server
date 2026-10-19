package gateway

import (
	"studio-engine/internal/pyjson"
)

// gatewaySettings are gateway_rules.DEFAULT_SETTINGS and RANGES per protocol.
var gatewaySettings = map[string]settingsRules{
	"iec104": gatewayRules(setting{"port", int64(2404)}, setting{"common_address", int64(1)},
		setting{"t1", int64(15)}, setting{"t2", int64(10)}, setting{"t3", int64(20)}, setting{"k", int64(12)},
		setting{"w", int64(8)}),
	"modbus": gatewayRules(setting{"port", int64(502)}, setting{"unit_id", int64(1)}),
	"dnp3": gatewayRules(setting{"port", int64(20000)}, setting{"outstation_address", int64(10)},
		setting{"master_address", int64(1)}, setting{"unsolicited", true}),
}

var gatewayRanges = map[string]limits{"port": {1, 65535}, "common_address": {1, 65534}, "t1": {1, 255},
	"t2": {1, 255}, "t3": {1, 3600}, "k": {1, 32767}, "w": {1, 32767}, "unit_id": {0, 255},
	"outstation_address": {0, 65519}, "master_address": {0, 65519}}

func gatewayRules(defaults ...setting) settingsRules {
	return settingsRules{defaults: defaults, ranges: gatewayRanges, flagText: func(key string) (string, bool) {
		return "Unsolicited reporting is on or off", key == "unsolicited"
	}}
}

// AddressRange is the inclusive point address range per gateway protocol.
var AddressRange = map[string][2]int64{"iec104": {1, 16777215}, "modbus": {0, 65535}, "dnp3": {0, 65535}}

var listen = map[string]string{"this_pc": "127.0.0.1", "network": "0.0.0.0"}

// EmptyGateway is empty_gateway().
func EmptyGateway() *pyjson.Object {
	settings := pyjson.NewObject()
	points := pyjson.NewObject()
	for _, name := range Protocols {
		settings.Set(name, gatewaySettings[name].object())
		points.Set(name, []any{})
	}
	return pyjson.NewObject().Set("role", "gateway").Set("master", EmptyMaster()).
		Set("protocol", "iec104").Set("listen", "this_pc").Set("real_commands", true).
		Set("auto_start", false).Set("settings", settings).Set("points", points)
}

// ValidateSettings is gateway_rules.validate_settings(protocol, values).
func ValidateSettings(protocol string, values any) (*pyjson.Object, error) {
	rules, ok := gatewaySettings[protocol]
	if !ok {
		return nil, valueError("Choose IEC 104, Modbus TCP or DNP3")
	}
	return rules.check(protocol, values)
}

// ListenAddress is listen_address(choice): the bind address of a listen choice.
func ListenAddress(choice any) (string, error) {
	if err := hashable(choice); err != nil {
		return "", err
	}
	text, isText := choice.(string)
	address, ok := listen[text]
	if !ok || !isText {
		return "", valueError("Choose 'This PC only' or 'All network adapters'")
	}
	return address, nil
}
