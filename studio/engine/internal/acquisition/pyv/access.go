// Package pyv reads and builds pyjson trees with Python dict/list semantics, so the live
// acquisition code can follow the Python source line by line (Go engine stream D).
package pyv

import "studio-engine/internal/pyjson"

// O builds an ordered object from key, value pairs: O("a", 1, "b", 2) is {"a": 1, "b": 2}.
func O(pairs ...any) *pyjson.Object {
	object := pyjson.NewObject()
	for index := 0; index+1 < len(pairs); index += 2 {
		object.Set(pairs[index].(string), pairs[index+1])
	}
	return object
}

// Get returns object.get(key): nil when the object is nil or the key is missing.
func Get(object *pyjson.Object, key string) any {
	if object == nil {
		return nil
	}
	value, _ := object.Get(key)
	return value
}

// Has reports `key in object`.
func Has(object *pyjson.Object, key string) bool {
	if object == nil {
		return false
	}
	_, found := object.Get(key)
	return found
}

// GetOr returns object.get(key, fallback): fallback only when the key is missing.
func GetOr(object *pyjson.Object, key string, fallback any) any {
	if object == nil {
		return fallback
	}
	if value, found := object.Get(key); found {
		return value
	}
	return fallback
}

// Obj returns value as an object, or nil when it is not one.
func Obj(value any) *pyjson.Object {
	object, _ := value.(*pyjson.Object)
	return object
}

// List returns value as a list, or nil when it is not one.
func List(value any) []any {
	items, _ := value.([]any)
	return items
}

// Str returns value as a string, or "" when it is not one.
func Str(value any) string {
	text, _ := value.(string)
	return text
}

// GetObj is Obj(Get(object, key)).
func GetObj(object *pyjson.Object, key string) *pyjson.Object { return Obj(Get(object, key)) }

// GetList is List(Get(object, key)).
func GetList(object *pyjson.Object, key string) []any { return List(Get(object, key)) }

// GetStr is Str(Get(object, key)).
func GetStr(object *pyjson.Object, key string) string { return Str(Get(object, key)) }

// OrObj is `object.get(key) or {}`: an empty object when the value is missing or falsy.
func OrObj(object *pyjson.Object, key string) *pyjson.Object {
	if found := GetObj(object, key); found != nil {
		return found
	}
	return pyjson.NewObject()
}

// Objects returns the object items of a list (non-objects are skipped).
func Objects(items []any) []*pyjson.Object {
	result := make([]*pyjson.Object, 0, len(items))
	for _, item := range items {
		if object, ok := item.(*pyjson.Object); ok {
			result = append(result, object)
		}
	}
	return result
}

// Strings converts a list of strings to []any.
func Strings(items []string) []any {
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return result
}

// Update sets every key of source on target, like dict.update.
func Update(target, source *pyjson.Object) *pyjson.Object {
	for _, key := range source.Keys() {
		value, _ := source.Get(key)
		target.Set(key, value)
	}
	return target
}

// Merge returns {**a, **b} as a new object.
func Merge(objects ...*pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, object := range objects {
		if object != nil {
			Update(result, object)
		}
	}
	return result
}
