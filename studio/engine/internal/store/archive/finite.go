package archive

import (
	"math"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// checkFinite mirrors json.dumps(allow_nan=False): the first NaN or infinity in encoding
// order raises CPython 3.12's ValueError "Out of range float values are not JSON
// compliant: <repr>" (repr is nan, inf or -inf).
func checkFinite(value any) error {
	switch v := value.(type) {
	case float64:
		switch {
		case math.IsNaN(v):
			return outOfRange("nan")
		case math.IsInf(v, 1):
			return outOfRange("inf")
		case math.IsInf(v, -1):
			return outOfRange("-inf")
		}
	case []any:
		for _, item := range v {
			if err := checkFinite(item); err != nil {
				return err
			}
		}
	case *pyjson.Object:
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			if err := checkFinite(item); err != nil {
				return err
			}
		}
	}
	return nil
}

func outOfRange(repr string) error {
	return apperr.ValueError("Out of range float values are not JSON compliant: " + repr)
}

// CheckFinite reports the error json.dumps(value, allow_nan=False) would raise, or nil.
func CheckFinite(value any) error { return checkFinite(value) }
