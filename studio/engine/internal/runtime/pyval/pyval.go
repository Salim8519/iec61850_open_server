// Package pyval gives JSON values (pyjson's nil, bool, int64, *big.Int, float64, string, []any,
// *pyjson.Object) the Python semantics the runtime relies on: ==, str(), truthiness and the
// isinstance checks with their bool quirks (True is an int in Python).
package pyval

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
)

// Obj builds an ordered object from key, value pairs (keys in call order, as a Python
// keyword-argument dict).
func Obj(pairs ...any) *pyjson.Object {
	object := pyjson.NewObject()
	for i := 0; i+1 < len(pairs); i += 2 {
		object.Set(pairs[i].(string), pairs[i+1])
	}
	return object
}

// Get returns object[key] or nil (dict.get).
func Get(object *pyjson.Object, key string) any {
	if object == nil {
		return nil
	}
	value, _ := object.Get(key)
	return value
}

// Str returns object[key] when it is a string, else "".
func Str(object *pyjson.Object, key string) string {
	text, _ := Get(object, key).(string)
	return text
}

// Map returns object[key] when it is an object.
func Map(object *pyjson.Object, key string) *pyjson.Object {
	child, _ := Get(object, key).(*pyjson.Object)
	return child
}

// List returns object[key] when it is a list.
func List(object *pyjson.Object, key string) []any {
	items, _ := Get(object, key).([]any)
	return items
}

// IsBool is isinstance(value, bool).
func IsBool(value any) bool { _, ok := value.(bool); return ok }

// IsInt is isinstance(value, int) and not bool.
func IsInt(value any) bool {
	switch value.(type) {
	case int64, int, *big.Int:
		return true
	}
	return false
}

// IsNumber is isinstance(value, (int, float)) and not bool.
func IsNumber(value any) bool {
	_, isFloat := value.(float64)
	return isFloat || IsInt(value)
}

// Float converts a number (bool counts, as in Python) to float64.
func Float(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case int64:
		return float64(typed), true
	case int:
		return float64(typed), true
	case bool:
		if typed {
			return 1, true
		}
		return 0, true
	case *big.Int:
		f, _ := new(big.Float).SetInt(typed).Float64()
		return f, true
	}
	return 0, false
}

// Equal is Python ==: numbers compare by value across bool/int/float, containers deeply.
func Equal(a, b any) bool {
	if fa, ok := numeric(a); ok {
		if fb, ok := numeric(b); ok {
			ia, aInt := intOf(a)
			ib, bInt := intOf(b)
			if aInt && bInt {
				return ia.Cmp(ib) == 0
			}
			return fa == fb
		}
		return false
	}
	switch typed := a.(type) {
	case nil:
		return b == nil
	case string:
		other, ok := b.(string)
		return ok && typed == other
	case []any:
		other, ok := b.([]any)
		if !ok || len(other) != len(typed) {
			return false
		}
		for i := range typed {
			if !Equal(typed[i], other[i]) {
				return false
			}
		}
		return true
	case *pyjson.Object:
		other, ok := b.(*pyjson.Object)
		if !ok || other.Len() != typed.Len() {
			return false
		}
		for _, key := range typed.Keys() {
			value, present := other.Get(key)
			mine, _ := typed.Get(key)
			if !present || !Equal(mine, value) {
				return false
			}
		}
		return true
	}
	return false
}

func numeric(value any) (float64, bool) {
	switch value.(type) {
	case bool, int64, int, float64, *big.Int:
		return Float(value)
	}
	return 0, false
}

func intOf(value any) (*big.Int, bool) {
	switch typed := value.(type) {
	case bool:
		if typed {
			return big.NewInt(1), true
		}
		return big.NewInt(0), true
	case int64:
		return big.NewInt(typed), true
	case int:
		return big.NewInt(int64(typed)), true
	case *big.Int:
		return typed, true
	}
	return nil, false
}

// Truthy is bool(value).
func Truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case []any:
		return len(typed) > 0
	case *pyjson.Object:
		return typed.Len() > 0
	}
	f, ok := Float(value)
	return !ok || f != 0
}

// Repr is str() of a value: str of strings, repr of floats, True/False/None, and Python's
// repr form for lists and dicts.
func Repr(value any) string {
	switch typed := value.(type) {
	case nil:
		return "None"
	case string:
		return typed
	case bool:
		if typed {
			return "True"
		}
		return "False"
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	case *big.Int:
		return typed.String()
	case float64:
		switch {
		case math.IsNaN(typed):
			return "nan"
		case math.IsInf(typed, 1):
			return "inf"
		case math.IsInf(typed, -1):
			return "-inf"
		}
		return pyjson.Repr(typed)
	case []any:
		parts := make([]string, len(typed))
		for i, item := range typed {
			parts[i] = inner(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *pyjson.Object:
		parts := make([]string, 0, typed.Len())
		for _, key := range typed.Keys() {
			item, _ := typed.Get(key)
			parts = append(parts, inner(key)+": "+inner(item))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return ""
}

func inner(value any) string {
	if text, ok := value.(string); ok {
		return "'" + strings.ReplaceAll(strings.ReplaceAll(text, `\`, `\\`), "'", `\'`) + "'"
	}
	return Repr(value)
}

// TypeName is type(value).__name__.
func TypeName(value any) string {
	switch value.(type) {
	case nil:
		return "NoneType"
	case bool:
		return "bool"
	case string:
		return "str"
	case float64:
		return "float"
	case []any:
		return "list"
	case *pyjson.Object:
		return "dict"
	}
	return "int"
}

// Copy returns a deep copy of a JSON value (copy.deepcopy / dict(...) of nested data).
func Copy(value any) any {
	switch typed := value.(type) {
	case []any:
		items := make([]any, len(typed))
		for i, item := range typed {
			items[i] = Copy(item)
		}
		return items
	case *pyjson.Object:
		object := pyjson.NewObject()
		for _, key := range typed.Keys() {
			item, _ := typed.Get(key)
			object.Set(key, Copy(item))
		}
		return object
	}
	return value
}
