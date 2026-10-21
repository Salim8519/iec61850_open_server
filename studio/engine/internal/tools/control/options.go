// Package control is tools/control_client.py: one warm studio_control worker operates IEC
// 61850 controls for IED Explorer and the gateway, one command at a time.
package control

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// orCategories maps the originator words to orCat values.
var orCategories = map[string]int64{"bay": 1, "station": 2, "remote": 3, "maintenance": 7}

// Options is control_options: {"or_cat", "interlock_check", "synchro_check", "test"}.
// options is the page's object (nil = {}); a non-object answers like Python's
// AttributeError (a plain error, HTTP 500).
func Options(options any) (*pyjson.Object, error) {
	object, ok := options.(*pyjson.Object)
	if options == nil || (!ok && !pytext.Truthy(options)) {
		object = pyjson.NewObject()
	} else if !ok {
		return nil, errAttribute("get")
	}
	category := any("remote")
	if value, found := object.Get("originator"); found {
		category = value
	}
	name, isText := category.(string)
	code, known := orCategories[name]
	if !isText || !known {
		if !hashable(category) {
			return nil, errUnhashable(category)
		}
		return nil, apperr.ValueError("Originator must be bay, station, remote or maintenance control")
	}
	result := pyjson.NewObject().Set("or_cat", code)
	for _, key := range []string{"interlock_check", "synchro_check", "test"} {
		value, found := object.Get(key)
		if !found {
			value = key == "interlock_check"
		}
		flag, isBool := value.(bool)
		if !isBool {
			return nil, apperr.ValueError(strings.ReplaceAll(key, "_", " ") + " is on or off")
		}
		result.Set(key, flag)
	}
	return result, nil
}

// Value is control_value: a bool, or an int that fits int32.
func Value(value any) (any, error) {
	if _, isBool := value.(bool); isBool || pytext.IntIn(value, -2147483648, 2147483647) {
		return value, nil
	}
	return nil, apperr.ValueError("A control value is on/off (true/false) or a whole number step")
}

// Summary is one plain sentence for logs and the page.
func Summary(result *pyjson.Object) string {
	ok, _ := result.Get("ok")
	if pytext.Truthy(ok) {
		how := "operated"
		if selected, _ := result.Get("selected"); pytext.Truthy(selected) {
			how = "selected and operated"
		}
		model, _ := result.Get("model")
		return "The device " + how + " it (" + pytext.Str(model) + ")."
	}
	if text, _ := result.Get("error"); pytext.Truthy(text) {
		return pytext.Str(text)
	}
	return "The device refused the command."
}

func hashable(value any) bool {
	switch value.(type) {
	case []any, *pyjson.Object:
		return false
	}
	return true
}

// pythonError is an exception Python did not map (TypeError, AttributeError): HTTP 500.
type pythonError struct{ text string }

func (e *pythonError) Error() string { return e.text }

func errAttribute(name string) error {
	return &pythonError{"AttributeError: object has no attribute '" + name + "'"}
}

func errUnhashable(value any) error {
	kind := "list"
	if _, isDict := value.(*pyjson.Object); isDict {
		kind = "dict"
	}
	return &pythonError{"TypeError: unhashable type: '" + kind + "'"}
}
