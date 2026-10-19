// Package signal holds the pure value rules of simulated signals (domain/signal.py, sample.py,
// source.py, csv_trace.py) and the source generators (simulation/generators.py), with Python's
// texts and number semantics.
package signal

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/runtime/pyval"
)

// Signal is the part of an imported signal dict the runtime reads.
type Signal struct {
	Ref, Kind, FC            string
	Writable                 bool
	QualityRef, TimestampRef string
}

// Validate is domain/signal.validate_value: it returns the value Python would send to the
// worker (FLOAT kinds become float) or a ValueError.
func Validate(kind string, value any) (any, error) {
	switch {
	case kind == "BOOLEAN":
		if !pyval.IsBool(value) {
			return nil, apperr.ValueError("Boolean signals require true or false")
		}
		return value, nil
	case strings.HasPrefix(kind, "FLOAT"):
		number, ok := pyval.Float(value)
		if pyval.IsBool(value) || !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, apperr.ValueError("Measurement must be a finite number")
		}
		if kind == "FLOAT32" && math.Abs(number) > 3.4028235e38 {
			return nil, apperr.ValueError("Measurement exceeds FLOAT32 range")
		}
		return number, nil
	case strings.HasPrefix(kind, "INT") || kind == "Enum" || kind == "Dbpos" || kind == "Tcmd" ||
		kind == "Quality" || kind == "Timestamp":
		if !pyval.IsInt(value) {
			return nil, apperr.ValueError("This signal requires an integer")
		}
		minimum, maximum := intRange(kind)
		number := toBig(value)
		if number.Cmp(minimum) < 0 || number.Cmp(maximum) > 0 {
			return nil, apperr.ValueError(fmt.Sprintf("Value must be between %s and %s", minimum, maximum))
		}
		return value, nil
	case strings.Contains(kind, "String") || kind == "ObjRef" || kind == "Currency":
		size := 129
		if strings.Contains(kind, "String") {
			parts := strings.Split(kind, "String")
			parsed, err := strconv.Atoi(parts[len(parts)-1])
			if err != nil {
				return nil, apperr.ValueError(fmt.Sprintf("invalid literal for int() with base 10: '%s'", parts[len(parts)-1]))
			}
			size = parsed
		}
		text, ok := value.(string)
		if !ok || len(text) > size {
			return nil, apperr.ValueError(fmt.Sprintf("Text must be at most %d UTF-8 bytes", size))
		}
		return value, nil
	}
	return nil, apperr.ValueError(fmt.Sprintf("Manual editing of %s is not implemented", kind))
}

func intRange(kind string) (*big.Int, *big.Int) {
	one := big.NewInt(1)
	shift := func(n uint) *big.Int { return new(big.Int).Lsh(one, n) }
	switch {
	case strings.HasPrefix(kind, "INT"):
		unsigned := strings.HasSuffix(kind, "U")
		width, _ := strconv.Atoi(strings.TrimRight(kind[3:], "U"))
		if unsigned {
			return big.NewInt(0), new(big.Int).Sub(shift(uint(width)), one)
		}
		return new(big.Int).Neg(shift(uint(width - 1))), new(big.Int).Sub(shift(uint(width-1)), one)
	case kind == "Dbpos" || kind == "Tcmd":
		return big.NewInt(0), big.NewInt(3)
	case kind == "Quality":
		return big.NewInt(0), big.NewInt(8191)
	case kind == "Timestamp":
		return big.NewInt(0), big.NewInt(4294967295999)
	}
	return new(big.Int).Neg(shift(31)), new(big.Int).Sub(shift(63), one)
}

func toBig(value any) *big.Int {
	switch typed := value.(type) {
	case int64:
		return big.NewInt(typed)
	case int:
		return big.NewInt(int64(typed))
	case *big.Int:
		return typed
	}
	return big.NewInt(0)
}
