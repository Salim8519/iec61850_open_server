// Package value validates engineering scalar values exactly as domain/signal.py
// validate_value does; it is pure and shared by the SCL exporter and project validation.
package value

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Validate checks value against signal (an imported signal object with "kind") and
// returns the normalised value (floats for FLOAT kinds) or a ValueError with Python's text.
func Validate(signal *pyjson.Object, v any) (any, error) {
	raw, _ := signal.Get("kind")
	kind, _ := raw.(string)
	return ValidateKind(kind, v)
}

// ValidateKind is Validate for a known basic type name.
func ValidateKind(kind string, v any) (any, error) {
	switch {
	case kind == "BOOLEAN":
		if b, ok := v.(bool); ok {
			return b, nil
		}
		return nil, apperr.ValueError("Boolean signals require true or false")
	case strings.HasPrefix(kind, "FLOAT"):
		return validateFloat(kind, v)
	case strings.HasPrefix(kind, "INT") || kind == "Enum" || kind == "Dbpos" || kind == "Tcmd" ||
		kind == "Quality" || kind == "Timestamp":
		return validateInt(kind, v)
	case strings.Contains(kind, "String") || kind == "ObjRef" || kind == "Currency":
		size := 129
		if strings.Contains(kind, "String") {
			parts := strings.Split(kind, "String")
			size, _ = strconv.Atoi(parts[len(parts)-1])
		}
		text, ok := v.(string)
		if !ok || len(text) > size {
			return nil, apperr.ValueError("Text must be at most " + strconv.Itoa(size) + " UTF-8 bytes")
		}
		return text, nil
	}
	return nil, apperr.ValueError("Manual editing of " + kind + " is not implemented")
}

func validateFloat(kind string, v any) (any, error) {
	var number float64
	switch n := v.(type) {
	case int:
		number = float64(n)
	case int64:
		number = float64(n)
	case *big.Int:
		f, _ := new(big.Float).SetInt(n).Float64()
		number = f
	case float64:
		number = n
	default:
		return nil, apperr.ValueError("Measurement must be a finite number")
	}
	if math.IsInf(number, 0) || math.IsNaN(number) {
		return nil, apperr.ValueError("Measurement must be a finite number")
	}
	if kind == "FLOAT32" && math.Abs(number) > 3.4028235e38 {
		return nil, apperr.ValueError("Measurement exceeds FLOAT32 range")
	}
	return number, nil
}

func validateInt(kind string, v any) (any, error) {
	var number *big.Int
	switch n := v.(type) {
	case int:
		number = big.NewInt(int64(n))
	case int64:
		number = big.NewInt(n)
	case *big.Int:
		number = n
	default:
		return nil, apperr.ValueError("This signal requires an integer")
	}
	minimum, maximum := bounds(kind)
	if number.Cmp(minimum) < 0 || number.Cmp(maximum) > 0 {
		return nil, apperr.ValueError("Value must be between " + minimum.String() + " and " + maximum.String())
	}
	if number.IsInt64() {
		return number.Int64(), nil
	}
	return number, nil
}

func bounds(kind string) (*big.Int, *big.Int) {
	one := big.NewInt(1)
	shifted := func(bits uint) *big.Int { return new(big.Int).Lsh(one, bits) }
	switch {
	case strings.HasPrefix(kind, "INT"):
		unsigned := strings.HasSuffix(kind, "U")
		width, _ := strconv.Atoi(strings.TrimRight(kind[3:], "U"))
		if unsigned {
			return big.NewInt(0), new(big.Int).Sub(shifted(uint(width)), one)
		}
		return new(big.Int).Neg(shifted(uint(width - 1))), new(big.Int).Sub(shifted(uint(width-1)), one)
	case kind == "Dbpos" || kind == "Tcmd":
		return big.NewInt(0), big.NewInt(3)
	case kind == "Quality":
		return big.NewInt(0), big.NewInt(8191)
	case kind == "Timestamp":
		return big.NewInt(0), big.NewInt(4294967295999)
	}
	return new(big.Int).Neg(shifted(31)), new(big.Int).Sub(shifted(63), one)
}
