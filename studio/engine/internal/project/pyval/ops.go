package pyval

import (
	"unicode/utf8"

	"studio-engine/internal/pyjson"
)

// Get is v.get(key, fallback); a non-dict raises AttributeError like Python.
func Get(v any, key string, fallback any) (any, error) {
	object, ok := v.(*pyjson.Object)
	if !ok {
		return nil, NoAttribute(v, "get")
	}
	if value, found := object.Get(key); found {
		return value, nil
	}
	return fallback, nil
}

// Lookup is dict.get on a known dict: the value or fallback.
func Lookup(object *pyjson.Object, key string, fallback any) any {
	if value, found := object.Get(key); found {
		return value
	}
	return fallback
}

// Has reports `key in object`.
func Has(object *pyjson.Object, key string) bool {
	_, found := object.Get(key)
	return found
}

// Pop is v.pop(key, fallback): a dict removes the key, a list raises TypeError (two
// arguments) and other types raise AttributeError.
func Pop(v any, key string, fallback any) (any, error) {
	switch x := v.(type) {
	case *pyjson.Object:
		value, found := x.Get(key)
		if !found {
			return fallback, nil
		}
		x.Delete(key)
		return value, nil
	case []any:
		return nil, TypeError("pop expected at most 1 argument, got 2")
	}
	return nil, NoAttribute(v, "pop")
}

// Items checks that v.items() exists (v is a dict) and returns it.
func Items(v any) (*pyjson.Object, error) {
	object, ok := v.(*pyjson.Object)
	if !ok {
		return nil, NoAttribute(v, "items")
	}
	return object, nil
}

// Iter returns what `for item in v` yields: list items, dict keys or one-character
// strings; numbers, bools and None raise TypeError.
func Iter(v any) ([]any, error) {
	switch x := v.(type) {
	case []any:
		return x, nil
	case *pyjson.Object:
		keys := x.Keys()
		items := make([]any, len(keys))
		for index, key := range keys {
			items[index] = key
		}
		return items, nil
	case string:
		items := make([]any, 0, utf8.RuneCountInString(x))
		for _, r := range x {
			items = append(items, string(r))
		}
		return items, nil
	}
	return nil, NotIterable(v)
}

// Merge is the dict display `{**a, **b, ...}`: later keys win, first positions stay.
func Merge(parts ...any) (*pyjson.Object, error) {
	result := pyjson.NewObject()
	for _, part := range parts {
		object, ok := part.(*pyjson.Object)
		if !ok {
			return nil, NotMapping(part)
		}
		for _, key := range object.Keys() {
			value, _ := object.Get(key)
			result.Set(key, value)
		}
	}
	return result, nil
}

// InStrings is `v in {"a", "b"}` (a set or dict of str): v is hashed first, so lists and
// dicts raise TypeError; only an equal str is a member.
func InStrings(v any, members ...string) (bool, error) {
	if err := CheckHashable(v); err != nil {
		return false, err
	}
	return InTuple(v, members...), nil
}

// InTuple is `v in ("a", "b")`: compared with == only, never hashed.
func InTuple(v any, members ...string) bool {
	text, ok := v.(string)
	if !ok {
		return false
	}
	for _, member := range members {
		if text == member {
			return true
		}
	}
	return false
}

// InValues is `v in container` for a list (compared with ==).
func InValues(v any, container []any) bool {
	for _, item := range container {
		if Equal(v, item) {
			return true
		}
	}
	return false
}

// Equal is Python's == for decoded JSON values (1 == 1.0 == True).
func Equal(a, b any) bool {
	if order, ok := NumberCompare(a, b); ok {
		return order == 0
	}
	switch x := a.(type) {
	case nil:
		return b == nil
	case string:
		y, ok := b.(string)
		return ok && x == y
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for index := range x {
			if !Equal(x[index], y[index]) {
				return false
			}
		}
		return true
	case *pyjson.Object:
		y, ok := b.(*pyjson.Object)
		if !ok || x.Len() != y.Len() {
			return false
		}
		for _, key := range x.Keys() {
			other, found := y.Get(key)
			value, _ := x.Get(key)
			if !found || !Equal(value, other) {
				return false
			}
		}
		return true
	}
	return false
}

// DeepCopy is copy.deepcopy for decoded JSON values.
func DeepCopy(v any) any {
	switch x := v.(type) {
	case []any:
		items := make([]any, len(x))
		for index, item := range x {
			items[index] = DeepCopy(item)
		}
		return items
	case *pyjson.Object:
		return CopyObject(x)
	}
	return v
}

// CopyShallow is dict(object): a new object sharing the values.
func CopyShallow(object *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range object.Keys() {
		value, _ := object.Get(key)
		result.Set(key, value)
	}
	return result
}

// CopyObject deep-copies one object.
func CopyObject(object *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range object.Keys() {
		value, _ := object.Get(key)
		result.Set(key, DeepCopy(value))
	}
	return result
}
