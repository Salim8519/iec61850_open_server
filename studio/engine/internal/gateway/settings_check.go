package gateway

import (
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// setting is one default setting; slices keep Python's dict order.
type setting struct {
	key   string
	value any
}

// limits is an inclusive whole-number range.
type limits struct{ low, high int64 }

// settingsRules are the defaults, ranges and on/off flags of one protocol's settings.
type settingsRules struct {
	defaults []setting
	ranges   map[string]limits
	flagText func(key string) (string, bool) // the error text of an on/off setting
}

func (rules settingsRules) object() *pyjson.Object {
	result := pyjson.NewObject()
	for _, item := range rules.defaults {
		result.Set(item.key, item.value)
	}
	return result
}

// settingLabel is key.replace("_", " ").capitalize() for the ASCII setting names.
func settingLabel(key string) string { return pytext.Capitalize(strings.ReplaceAll(key, "_", " ")) }

// check is the validate_settings loop shared by gateway_rules and master_rules.
func (rules settingsRules) check(protocol string, values any) (*pyjson.Object, error) {
	result := rules.object()
	items, err := pairs(values)
	if err != nil {
		return nil, err
	}
	for _, key := range items.Keys() {
		value, _ := items.Get(key)
		if _, known := result.Get(key); !known {
			return nil, valueError("Unknown " + protocol + " setting '" + key + "'")
		}
		if text, flag := rules.flagText(key); flag {
			if _, ok := value.(bool); !ok {
				return nil, valueError(text)
			}
		} else {
			bounds := rules.ranges[key]
			if !pytext.IntIn(value, bounds.low, bounds.high) {
				return nil, valueError(settingLabel(key) + " must be a whole number from " +
					strconv.FormatInt(bounds.low, 10) + " to " + strconv.FormatInt(bounds.high, 10))
			}
		}
		result.Set(key, value)
	}
	if protocol == "iec104" {
		if !(number(result, "t2") < number(result, "t1")) {
			return nil, valueError("t2 must be shorter than t1 (IEC 60870-5-104)")
		}
		if !(number(result, "w") <= number(result, "k")) {
			return nil, valueError("w must not be larger than k (IEC 60870-5-104)")
		}
	}
	return result, nil
}

// number reads a validated whole-number setting.
func number(object *pyjson.Object, key string) int64 {
	value, _ := pytext.Int64(get(object, key))
	return value
}
