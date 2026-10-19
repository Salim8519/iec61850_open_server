package pyv

import (
	"math"
	"math/big"

	"studio-engine/internal/pyjson"
)

// Truthy is Python's bool(value) for pyjson values.
func Truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case *big.Int:
		return v.Sign() != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case *pyjson.Object:
		return v.Len() > 0
	}
	return true
}

// IsInt is isinstance(value, int) and not isinstance(value, bool).
func IsInt(value any) bool {
	switch value.(type) {
	case int, int64, *big.Int:
		return true
	}
	return false
}

// Int64 returns an integer value (bool excluded) that fits in int64.
func Int64(value any) (int64, bool) {
	switch v := value.(type) {
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

// number returns a numeric value as big.Float and whether it is a number; bool counts as
// 0/1 like Python's int subclass.
func number(value any) (*big.Float, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return big.NewFloat(1), true
		}
		return big.NewFloat(0), true
	case int:
		return new(big.Float).SetInt64(int64(v)), true
	case int64:
		return new(big.Float).SetInt64(v), true
	case *big.Int:
		return new(big.Float).SetInt(v), true
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, false
		}
		return big.NewFloat(v), true
	}
	return nil, false
}

// Equal is Python's == for pyjson values (1 == 1.0 == True; NaN never equal).
func Equal(a, b any) bool {
	if fa, ok := a.(float64); ok && (math.IsNaN(fa) || math.IsInf(fa, 0)) {
		fb, isFloat := b.(float64)
		return isFloat && fa == fb
	}
	if fb, ok := b.(float64); ok && (math.IsNaN(fb) || math.IsInf(fb, 0)) {
		return false
	}
	if na, ok := number(a); ok {
		nb, isNumber := number(b)
		return isNumber && na.Cmp(nb) == 0
	}
	switch v := a.(type) {
	case nil:
		return b == nil
	case string:
		other, ok := b.(string)
		return ok && v == other
	case []any:
		other, ok := b.([]any)
		if !ok || len(v) != len(other) {
			return false
		}
		for index := range v {
			if !Equal(v[index], other[index]) {
				return false
			}
		}
		return true
	case *pyjson.Object:
		other, ok := b.(*pyjson.Object)
		if !ok || v.Len() != other.Len() {
			return false
		}
		for _, key := range v.Keys() {
			left, _ := v.Get(key)
			right, found := other.Get(key)
			if !found || !Equal(left, right) {
				return false
			}
		}
		return true
	}
	return false
}

// RoundHalfEven is Python's round(x) for a float: an int, ties to even.
func RoundHalfEven(x float64) int64 {
	return int64(math.RoundToEven(x))
}
