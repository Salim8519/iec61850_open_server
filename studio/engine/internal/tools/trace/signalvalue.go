// Package trace is CSV measurement playback (domain/csv_trace.py) and the manual value
// rule it relies on (domain/signal.py validate_value). Stream C owns the runtime sources
// that play a trace; this package owns parsing, validation and the value at a time.
package trace

import (
	"math"
	"math/big"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// ValidateValue is domain/signal.validate_value(signal, value).
func ValidateValue(signal *pyjson.Object, value any) (any, error) {
	kindValue, _ := signal.Get("kind")
	kind := pytext.Str(kindValue)
	switch {
	case kind == "BOOLEAN":
		if _, ok := value.(bool); !ok {
			return nil, apperr.ValueError("Boolean signals require true or false")
		}
		return value, nil
	case strings.HasPrefix(kind, "FLOAT"):
		_, isBool := value.(bool)
		number, isNumber := pytext.Float(value)
		if isBool || !isNumber || !pytext.IsNumber(value) || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, apperr.ValueError("Measurement must be a finite number")
		}
		if kind == "FLOAT32" && math.Abs(number) > 3.4028235e38 {
			return nil, apperr.ValueError("Measurement exceeds FLOAT32 range")
		}
		return number, nil
	case strings.HasPrefix(kind, "INT") || kind == "Enum" || kind == "Dbpos" || kind == "Tcmd" ||
		kind == "Quality" || kind == "Timestamp":
		if !pytext.IsInt(value) {
			return nil, apperr.ValueError("This signal requires an integer")
		}
		low, high, err := intRange(kind)
		if err != nil {
			return nil, err
		}
		number := toBig(value)
		if number.Cmp(low) < 0 || number.Cmp(high) > 0 {
			return nil, apperr.ValueError("Value must be between " + low.String() + " and " + high.String())
		}
		return value, nil
	case strings.Contains(kind, "String") || kind == "ObjRef" || kind == "Currency":
		size := int64(129)
		if strings.Contains(kind, "String") {
			parts := strings.Split(kind, "String")
			parsed, err := pytext.ParseInt(parts[len(parts)-1])
			if err != nil {
				return nil, apperr.ValueError(err.Error())
			}
			size, _ = pytext.Int64(parsed)
		}
		text, ok := value.(string)
		if !ok || int64(len(text)) > size {
			return nil, apperr.ValueError("Text must be at most " + big.NewInt(size).String() + " UTF-8 bytes")
		}
		return value, nil
	}
	return nil, apperr.ValueError("Manual editing of " + kind + " is not implemented")
}

func intRange(kind string) (*big.Int, *big.Int, error) {
	one := big.NewInt(1)
	shift := func(bits uint) *big.Int { return new(big.Int).Lsh(one, bits) }
	switch {
	case strings.HasPrefix(kind, "INT"):
		unsigned := strings.HasSuffix(kind, "U")
		parsed, err := pytext.ParseInt(strings.TrimRight(kind[3:], "U"))
		if err != nil {
			return nil, nil, apperr.ValueError(err.Error())
		}
		width, _ := pytext.Int64(parsed)
		if width < 0 || (!unsigned && width < 1) || width > 4096 {
			return nil, nil, apperr.ValueError("negative shift count") // no SCL basic type gets here
		}
		if unsigned {
			return big.NewInt(0), new(big.Int).Sub(shift(uint(width)), one), nil
		}
		low := new(big.Int).Neg(shift(uint(width - 1)))
		return low, new(big.Int).Sub(shift(uint(width-1)), one), nil
	case kind == "Dbpos" || kind == "Tcmd":
		return big.NewInt(0), big.NewInt(3), nil
	case kind == "Quality":
		return big.NewInt(0), big.NewInt(8191), nil
	case kind == "Timestamp":
		return big.NewInt(0), big.NewInt(4294967295999), nil
	}
	return new(big.Int).Neg(shift(31)), new(big.Int).Sub(shift(63), one), nil
}

func toBig(value any) *big.Int {
	if number, ok := value.(*big.Int); ok {
		return number
	}
	number, _ := pytext.Int64(value)
	return big.NewInt(number)
}
