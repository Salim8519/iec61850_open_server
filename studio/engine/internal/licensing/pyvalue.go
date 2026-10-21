package licensing

import (
	"errors"
	"math"
	"math/big"
	"strconv"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// errNotInt is the TypeError/ValueError of int(value); errIntOverflow is the
// OverflowError of int(inf), which Python's callers do not catch.
var (
	errNotInt      = errors.New("int() argument is not a number")
	errIntOverflow = errors.New("cannot convert float infinity to integer")
)

// Truthy is Python's bool(value) for decoded JSON values (shared with the update check).
func Truthy(value any) bool { return truthy(value) }

// truthy is Python's bool(value) for decoded JSON values.
func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int64:
		return v != 0
	case int:
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

// scalarText is str(value) for JSON scalars; ok is false for lists and objects, whose
// repr is never needed (it can never equal a machine ID or start with "SUBS-").
func scalarText(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "None", true
	case bool:
		if v {
			return "True", true
		}
		return "False", true
	case int64:
		return strconv.FormatInt(v, 10), true
	case int:
		return strconv.Itoa(v), true
	case *big.Int:
		return v.String(), true
	case float64:
		switch {
		case math.IsNaN(v):
			return "nan", true
		case math.IsInf(v, 1):
			return "inf", true
		case math.IsInf(v, -1):
			return "-inf", true
		}
		return pyjson.Repr(v), true
	case string:
		return v, true
	}
	return "", false
}

// saturate keeps a Python int inside int64; only absurd signed values reach the bounds,
// where every comparison the license rules make still gives Python's answer.
func saturate(value *big.Int) int64 {
	if value.IsInt64() {
		return value.Int64()
	}
	if value.Sign() < 0 {
		return math.MinInt64
	}
	return math.MaxInt64
}

// pyInt is int(value) for decoded JSON values.
func pyInt(value any) (int64, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case *big.Int:
		return saturate(v), nil
	case float64:
		if math.IsNaN(v) {
			return 0, errNotInt
		}
		if math.IsInf(v, 0) {
			return 0, errIntOverflow
		}
		whole, _ := new(big.Float).SetFloat64(math.Trunc(v)).Int(nil)
		return saturate(whole), nil
	case string:
		parsed, err := pytext.ParseInt(v)
		if err != nil {
			return 0, errNotInt
		}
		return saturate(parsed), nil
	}
	return 0, errNotInt
}

// lookup is dict.get(key) on a decoded JSON object.
func lookup(object *pyjson.Object, key string) (any, bool) {
	if object == nil {
		return nil, false
	}
	return object.Get(key)
}
