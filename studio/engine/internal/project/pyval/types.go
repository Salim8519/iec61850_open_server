// Package pyval reproduces the few CPython value rules the project code relies on when it
// handles decoded JSON (pyjson values): type names, truthiness, ==, membership with hashing,
// dict.get/pop, iteration, str()/repr(), float()/int() and the exception texts they raise.
// It is pure and imports only the standard library and the shared pyjson/pyrules packages.
package pyval

import (
	"fmt"
	"math/big"

	"studio-engine/internal/pyjson"
)

// TypeName returns type(v).__name__ for a decoded JSON value.
func TypeName(v any) string {
	switch v.(type) {
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
	return fmt.Sprintf("%T", v)
}

// Truthy is Python's bool(v).
func Truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case int:
		return x != 0
	case int64:
		return x != 0
	case *big.Int:
		return x.Sign() != 0
	case float64:
		return x != 0
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	case *pyjson.Object:
		return x != nil && x.Len() > 0
	}
	return true
}

// Or is Python's `v or fallback`.
func Or(v, fallback any) any {
	if Truthy(v) {
		return v
	}
	return fallback
}

// IsInt reports isinstance(v, int) and not isinstance(v, bool).
func IsInt(v any) bool {
	switch v.(type) {
	case int, int64, *big.Int:
		return true
	}
	return false
}

// IsIntOrBool reports isinstance(v, int), which is true for bool as well.
func IsIntOrBool(v any) bool {
	_, isBool := v.(bool)
	return isBool || IsInt(v)
}

// IsNumber reports isinstance(v, (int, float)) and not isinstance(v, bool).
func IsNumber(v any) bool {
	_, isFloat := v.(float64)
	return isFloat || IsInt(v)
}

// Dict returns v as an object when it is a dict.
func Dict(v any) (*pyjson.Object, bool) {
	object, ok := v.(*pyjson.Object)
	return object, ok
}

// List returns v as a list when it is a list.
func List(v any) ([]any, bool) {
	items, ok := v.([]any)
	return items, ok
}

// Hashable reports whether v can be hashed (lists and dicts cannot).
func Hashable(v any) bool {
	switch v.(type) {
	case []any, *pyjson.Object:
		return false
	}
	return true
}

// BigInt returns an int value (int, int64, *big.Int or bool) as a big.Int.
func BigInt(v any) (*big.Int, bool) {
	switch x := v.(type) {
	case bool:
		if x {
			return big.NewInt(1), true
		}
		return big.NewInt(0), true
	case int:
		return big.NewInt(int64(x)), true
	case int64:
		return big.NewInt(x), true
	case *big.Int:
		return x, true
	}
	return nil, false
}

// NormInt returns an integer as int64 when it fits, otherwise *big.Int (pyjson's form).
func NormInt(n *big.Int) any {
	if n.IsInt64() {
		return n.Int64()
	}
	return new(big.Int).Set(n)
}
