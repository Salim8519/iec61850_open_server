package netcfg

import (
	"math"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// itemFields reads change items like Python's item["key"]: the first missing key is kept
// and reported as KeyError, the way the Python code would have raised it.
type itemFields struct {
	item    *pyjson.Object
	missing string
	invalid string
}

func fieldsOf(item *pyjson.Object) *itemFields { return &itemFields{item: item} }

func (f *itemFields) value(key string) any {
	value, found := f.item.Get(key)
	if !found && f.missing == "" {
		f.missing = key
	}
	return value
}

// text is str(item[key]).
func (f *itemFields) text(key string) string { return pyStr(f.value(key)) }

// integer is str(int(item[key])).
func (f *itemFields) integer(key string) string {
	switch v := f.value(key).(type) {
	case bool:
		if v {
			return "1"
		}
		return "0"
	case float64:
		if !math.IsInf(v, 0) && !math.IsNaN(v) {
			return strconv.FormatFloat(math.Trunc(v), 'f', 0, 64)
		}
	case string:
		if number, err := strconv.ParseInt(strings.ReplaceAll(pyStrip(v), "_", ""), 10, 64); err == nil {
			return strconv.FormatInt(number, 10)
		}
	default:
		if number, ok := pyInt(v); ok {
			return number.String()
		}
	}
	if f.invalid == "" {
		f.invalid = key
	}
	return ""
}

// joined is sep.join(map(str, item[key])) for a list value.
func (f *itemFields) joined(key, sep string) string {
	list, _ := f.value(key).([]any)
	parts := make([]string, len(list))
	for index, value := range list {
		parts[index] = pyStr(value)
	}
	return strings.Join(parts, sep)
}

func (f *itemFields) err() error {
	if f.missing != "" {
		return apperr.KeyError(f.missing)
	}
	if f.invalid != "" {
		return apperr.ValueError("invalid literal for int() in " + f.invalid)
	}
	return nil
}

// ItemKey is journal.item_key: alias:<guid>:<ip>, unblock:<name> or firewall:<name>.
func ItemKey(item *pyjson.Object) (string, error) {
	f := fieldsOf(item)
	var key string
	switch f.text("kind") {
	case "alias":
		key = "alias:" + f.text("adapter_guid") + ":" + f.text("ip")
	case "unblock":
		key = "unblock:" + f.text("name")
	default:
		key = "firewall:" + f.text("name")
	}
	return key, f.err()
}
