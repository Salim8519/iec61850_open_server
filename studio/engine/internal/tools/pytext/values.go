package pytext

import (
	"math"
	"math/big"

	"studio-engine/internal/pyjson"
)

// IsInt is isinstance(value, int) and not isinstance(value, bool): a JSON integer.
func IsInt(value any) bool {
	switch value.(type) {
	case int, int64, *big.Int:
		return true
	}
	return false
}

// IsIntOrBool is isinstance(value, int) without a bool guard (True counts as 1).
func IsIntOrBool(value any) bool {
	_, isBool := value.(bool)
	return isBool || IsInt(value)
}

// IsNumber is isinstance(value, (int, float)) without a bool guard.
func IsNumber(value any) bool {
	_, isFloat := value.(float64)
	return isFloat || IsIntOrBool(value)
}

// IntIn reports whether value is a JSON integer (bool excluded) with low <= value <= high.
func IntIn(value any, low, high int64) bool {
	number, ok := Int64(value)
	return ok && IsInt(value) && number >= low && number <= high
}

// Int64 returns an integer or bool value as int64; ok is false for other values and for
// integers outside int64.
func Int64(value any) (int64, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case int:
		return int64(v), true
	case int64:
		return v, true
	case *big.Int:
		if v.IsInt64() {
			return v.Int64(), true
		}
	}
	return 0, false
}

// Float returns float(value) for an int, bool or float.
func Float(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case *big.Int:
		f, _ := new(big.Float).SetInt(v).Float64()
		return f, true
	}
	number, ok := Int64(value)
	return float64(number), ok
}

// Equal is Python's == between two values decoded by pyjson (1 == 1.0 == True).
func Equal(a, b any) bool {
	if IsNumber(a) && IsNumber(b) {
		return numberEqual(a, b)
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
			left, _ := x.Get(key)
			right, found := y.Get(key)
			if !found || !Equal(left, right) {
				return false
			}
		}
		return true
	}
	return false
}

func numberEqual(a, b any) bool {
	_, aFloat := a.(float64)
	_, bFloat := b.(float64)
	if !aFloat && !bFloat {
		return bigOf(a).Cmp(bigOf(b)) == 0
	}
	x, _ := Float(a)
	y, _ := Float(b)
	if math.IsNaN(x) || math.IsNaN(y) {
		return false
	}
	if aFloat && bFloat {
		return x == y
	}
	if math.IsInf(x, 0) || math.IsInf(y, 0) {
		return false
	}
	// int == float compares exactly in Python, not after rounding the int to a float.
	whole, exact := new(big.Float).SetFloat64(pick(aFloat, x, y)).Int(nil)
	if exact != big.Exact {
		return false
	}
	return whole.Cmp(bigOf(pick(aFloat, b, a))) == 0
}

func pick[T any](first bool, x, y T) T {
	if first {
		return x
	}
	return y
}

func bigOf(value any) *big.Int {
	if v, ok := value.(*big.Int); ok {
		return v
	}
	number, _ := Int64(value)
	return big.NewInt(number)
}
