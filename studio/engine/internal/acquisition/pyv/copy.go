package pyv

import (
	"math/big"

	"studio-engine/internal/pyjson"
)

// Copy returns a deep copy of a pyjson value (copy.deepcopy).
func Copy(value any) any {
	switch v := value.(type) {
	case []any:
		items := make([]any, len(v))
		for index, item := range v {
			items[index] = Copy(item)
		}
		return items
	case *pyjson.Object:
		return CopyObject(v)
	case *big.Int:
		return new(big.Int).Set(v)
	}
	return value
}

// CopyObject returns a deep copy of an object (nil stays nil).
func CopyObject(object *pyjson.Object) *pyjson.Object {
	if object == nil {
		return nil
	}
	result := pyjson.NewObject()
	for _, key := range object.Keys() {
		item, _ := object.Get(key)
		result.Set(key, Copy(item))
	}
	return result
}

// Shallow returns dict(object): a new object sharing the values.
func Shallow(object *pyjson.Object) *pyjson.Object {
	return Merge(object)
}
