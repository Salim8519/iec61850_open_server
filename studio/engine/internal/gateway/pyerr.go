// Package gateway is the Go port of the frozen gateway and SCADA master domain rules
// (studio_app/domain gateway_points, gateway_selection, gateway_rules, gateway_csv,
// passthrough, master_rules, master_points). Results and error texts are byte-identical to
// Python; values are pyjson values and Python dicts are *pyjson.Object in insertion order.
//
// A Python ValueError is returned as *apperr.Error (HTTP 400). Any other Python exception
// (TypeError, KeyError, AttributeError, OverflowError, csv.Error) is returned as a plain
// error whose text is "<Class>: <message>", so the HTTP layer answers 500 as Flask did.
package gateway

import (
	"math/big"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Crash is a Python exception other than ValueError; Flask answered it with 500.
type Crash struct {
	Class string
	Text  string
}

func (e *Crash) Error() string { return e.Class + ": " + e.Text }

func crash(class, text string) error { return &Crash{Class: class, Text: text} }

func valueError(text string) error { return apperr.ValueError(text) }

// typeName is type(value).__name__ for a pyjson value.
func typeName(value any) string {
	switch value.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case int, int64, *big.Int:
		return "int"
	case float64:
		return "float"
	case string:
		return "str"
	case []any:
		return "list"
	case *pyjson.Object:
		return "dict"
	}
	return "object"
}

// keyError is KeyError(key) for a missing dict key; Python prints repr(key).
func keyError(key any) error { return crash("KeyError", pytext.Repr(key)) }

func noAttribute(value any, name string) error {
	return crash("AttributeError", "'"+typeName(value)+"' object has no attribute '"+name+"'")
}

// hashable reports a TypeError for list and dict values used as dict keys or set members.
func hashable(value any) error {
	switch value.(type) {
	case []any, *pyjson.Object:
		return crash("TypeError", "unhashable type: '"+typeName(value)+"'")
	}
	return nil
}

// get is dict.get(key) (None when missing).
func get(object *pyjson.Object, key string) any {
	value, _ := object.Get(key)
	return value
}

// getOr is dict.get(key, fallback).
func getOr(object *pyjson.Object, key string, fallback any) any {
	if value, found := object.Get(key); found {
		return value
	}
	return fallback
}

// getAny is value.get(key, fallback) for any value: AttributeError when it is no dict.
func getAny(value any, key string, fallback any) (any, error) {
	object, ok := value.(*pyjson.Object)
	if !ok {
		return nil, noAttribute(value, "get")
	}
	return getOr(object, key, fallback), nil
}

// orGet is (value or {}).get(key, fallback).
func orGet(value any, key string, fallback any) (any, error) {
	if !pytext.Truthy(value) {
		return fallback, nil
	}
	return getAny(value, key, fallback)
}

// subscript is container[key] for a str key, with Python's error for each container type.
func subscript(container any, key string) (any, error) {
	switch v := container.(type) {
	case *pyjson.Object:
		if value, found := v.Get(key); found {
			return value, nil
		}
		return nil, keyError(key)
	case []any:
		return nil, crash("TypeError", "list indices must be integers or slices, not str")
	case string:
		return nil, crash("TypeError", "string indices must be integers, not 'str'")
	}
	return nil, crash("TypeError", "'"+typeName(container)+"' object is not subscriptable")
}

// iterate is iter(value): list items, dict keys or the characters of a str.
func iterate(value any) ([]any, error) {
	switch v := value.(type) {
	case []any:
		return v, nil
	case *pyjson.Object:
		keys := v.Keys()
		items := make([]any, len(keys))
		for index, key := range keys {
			items[index] = key
		}
		return items, nil
	case string:
		items := []any{}
		for _, r := range v {
			items = append(items, string(r))
		}
		return items, nil
	}
	return nil, crash("TypeError", "'"+typeName(value)+"' object is not iterable")
}

// pairs is (value or {}).items() as ordered keys and values.
func pairs(value any) (*pyjson.Object, error) {
	if !pytext.Truthy(value) {
		return pyjson.NewObject(), nil
	}
	object, ok := value.(*pyjson.Object)
	if !ok {
		return nil, noAttribute(value, "items")
	}
	return object, nil
}

// copyObject is dict(object): a new dict with the same keys in the same order.
func copyObject(object *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range object.Keys() {
		value, _ := object.Get(key)
		result.Set(key, value)
	}
	return result
}

// isStr reports isinstance(value, str) and returns the text.
func isStr(value any) (string, bool) {
	text, ok := value.(string)
	return text, ok
}

func contains(list []string, value any) bool {
	text, ok := value.(string)
	if !ok {
		return false
	}
	for _, item := range list {
		if item == text {
			return true
		}
	}
	return false
}

func anyStrings(list []string) []any {
	result := make([]any, len(list))
	for index, item := range list {
		result[index] = item
	}
	return result
}
