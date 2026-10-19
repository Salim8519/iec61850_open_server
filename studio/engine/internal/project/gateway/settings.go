// Package gateway normalises the stored gateway and SCADA master sections exactly as
// domain/gateway_rules.py and domain/master_rules.py do (ADR-0061).
package gateway

import (
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// setting is one protocol setting: a default and either a range or an on/off flag.
type setting struct {
	name      string
	fallback  any
	low, high int64
	flag      bool
}

func number(name string, fallback, low, high int64) setting {
	return setting{name: name, fallback: fallback, low: low, high: high}
}

func flag(name string, fallback bool) setting {
	return setting{name: name, fallback: fallback, flag: true}
}

func defaults(table []setting) *pyjson.Object {
	result := pyjson.NewObject()
	for _, item := range table {
		result.Set(item.name, item.fallback)
	}
	return result
}

// validateSettings is validate_settings of both rule modules: defaults overridden by the
// given values, every value checked; flagText words the on/off error for a flag key.
func validateSettings(protocol string, table []setting, values any, flagText func(string) string) (*pyjson.Object, error) {
	result := defaults(table)
	given, err := pyval.Items(pyval.Or(values, pyjson.NewObject()))
	if err != nil {
		return nil, err
	}
	for _, key := range given.Keys() {
		value, _ := given.Get(key)
		var item *setting
		for index := range table {
			if table[index].name == key {
				item = &table[index]
			}
		}
		if item == nil {
			return nil, apperr.ValueError("Unknown " + protocol + " setting '" + key + "'")
		}
		if item.flag {
			if _, ok := value.(bool); !ok {
				return nil, apperr.ValueError(flagText(key))
			}
		} else if !pyval.IsInt(value) || !pyval.Between(value, float64(item.low), float64(item.high)) {
			return nil, apperr.ValueError(settingLabel(key) + " must be a whole number from " +
				strconv.FormatInt(item.low, 10) + " to " + strconv.FormatInt(item.high, 10))
		}
		result.Set(key, value)
	}
	if protocol == "iec104" {
		t1, _ := result.Get("t1")
		t2, _ := result.Get("t2")
		if order, _ := pyval.NumberCompare(t2, t1); order >= 0 {
			return nil, apperr.ValueError("t2 must be shorter than t1 (IEC 60870-5-104)")
		}
		w, _ := result.Get("w")
		k, _ := result.Get("k")
		if order, _ := pyval.NumberCompare(w, k); order > 0 {
			return nil, apperr.ValueError("w must not be larger than k (IEC 60870-5-104)")
		}
	}
	return result, nil
}

// settingLabel is key.replace("_", " ").capitalize().
func settingLabel(key string) string {
	return pyval.Capitalize(strings.ReplaceAll(key, "_", " "))
}

// Station gateway settings per protocol (gateway_rules.DEFAULT_SETTINGS and RANGES).
var gatewaySettings = map[string][]setting{
	"iec104": {number("port", 2404, 1, 65535), number("common_address", 1, 1, 65534), number("t1", 15, 1, 255),
		number("t2", 10, 1, 255), number("t3", 20, 1, 3600), number("k", 12, 1, 32767), number("w", 8, 1, 32767)},
	"modbus": {number("port", 502, 1, 65535), number("unit_id", 1, 0, 255)},
	"dnp3": {number("port", 20000, 1, 65535), number("outstation_address", 10, 0, 65519),
		number("master_address", 1, 0, 65519), flag("unsolicited", true)},
}

// SCADA master settings per protocol (master_rules.DEFAULT_SETTINGS and RANGES).
var masterSettings = map[string][]setting{
	"iec104": {number("port", 2404, 1, 65535), number("common_address", 1, 1, 65534),
		number("originator_address", 0, 0, 255), number("t0", 10, 1, 255), number("t1", 15, 1, 255),
		number("t2", 10, 1, 255), number("t3", 20, 1, 3600), number("k", 12, 1, 32767), number("w", 8, 1, 32767),
		flag("time_tagged_commands", false), number("gi_minutes", 15, 0, 1440)},
	"dnp3": {number("port", 20000, 1, 65535), number("outstation_address", 10, 0, 65519),
		number("master_address", 1, 0, 65519), number("integrity_minutes", 15, 0, 1440),
		number("event_seconds", 5, 0, 3600), flag("unsolicited", true)},
	"modbus": {number("port", 502, 1, 65535), number("unit_id", 1, 0, 255), number("poll_ms", 1000, 200, 60000)},
}
