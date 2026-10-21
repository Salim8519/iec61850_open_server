package pyval

import (
	"math"
	"math/big"

	"studio-engine/internal/apperr"
	"studio-engine/internal/scl/pyrules"
)

// NumberCompare compares two numbers (int, float or bool) exactly as Python does and
// reports false when either is not a number or is NaN.
func NumberCompare(a, b any) (int, bool) {
	x, okA := numberRat(a)
	y, okB := numberRat(b)
	if !okA || !okB {
		return 0, false
	}
	if x.inf != 0 || y.inf != 0 {
		switch {
		case x.inf == y.inf:
			return 0, true
		case x.inf > y.inf:
			return 1, true
		}
		return -1, true
	}
	return x.value.Cmp(y.value), true
}

type exact struct {
	value *big.Rat
	inf   int
}

func numberRat(v any) (exact, bool) {
	if f, ok := v.(float64); ok {
		switch {
		case math.IsNaN(f):
			return exact{}, false
		case math.IsInf(f, 1):
			return exact{inf: 1}, true
		case math.IsInf(f, -1):
			return exact{inf: -1}, true
		}
		return exact{value: new(big.Rat).SetFloat64(f)}, true
	}
	if n, ok := BigInt(v); ok {
		return exact{value: new(big.Rat).SetInt(n)}, true
	}
	return exact{}, false
}

// Between is `low <= v <= high` for a number v (int or float); NaN is never between.
func Between(v any, low, high float64) bool {
	lower, ok := NumberCompare(v, low)
	upper, _ := NumberCompare(v, high)
	return ok && lower >= 0 && upper <= 0
}

// IntToFloat converts an int like Python's float(int): OverflowError beyond the range.
func IntToFloat(n *big.Int) (float64, error) {
	f, _ := new(big.Float).SetInt(n).Float64()
	if math.IsInf(f, 0) {
		return 0, OverflowError("int too large to convert to float")
	}
	return f, nil
}

// Float is Python's float(v) for a decoded JSON value.
func Float(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case string:
		f, err := pyrules.Float(x)
		if err != nil {
			return 0, apperr.ValueError(pyrules.FloatMessage(x))
		}
		return f, nil
	}
	if n, ok := BigInt(v); ok {
		return IntToFloat(n)
	}
	return 0, TypeError("float() argument must be a string or a real number, not '" + TypeName(v) + "'")
}

// Int is Python's int(v) for a decoded JSON value: int64 or *big.Int.
func Int(v any) (any, error) {
	switch x := v.(type) {
	case float64:
		switch {
		case math.IsNaN(x):
			return nil, apperr.ValueError("cannot convert float NaN to integer")
		case math.IsInf(x, 0):
			return nil, OverflowError("cannot convert float infinity to integer")
		}
		n, _ := new(big.Float).SetFloat64(math.Trunc(x)).Int(nil)
		return NormInt(n), nil
	case string:
		n, err := pyrules.IntBig(x, 10)
		if err != nil {
			return nil, apperr.ValueError(pyrules.IntMessage(x))
		}
		return NormInt(n), nil
	}
	if n, ok := BigInt(v); ok {
		return NormInt(n), nil
	}
	return nil, TypeError("int() argument must be a string, a bytes-like object or a real number, not '" +
		TypeName(v) + "'")
}

// IsFinite is math.isfinite(v) for an int or float (OverflowError for a huge int).
func IsFinite(v any) (bool, error) {
	if f, ok := v.(float64); ok {
		return !math.IsInf(f, 0) && !math.IsNaN(f), nil
	}
	if n, ok := BigInt(v); ok {
		if _, err := IntToFloat(n); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, TypeError("must be real number, not " + TypeName(v))
}

// AsFloat is float(v) for an int or float already known to be finite and in range.
func AsFloat(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	n, _ := BigInt(v)
	f, _ := IntToFloat(n)
	return f
}
