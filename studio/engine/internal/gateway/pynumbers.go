package gateway

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/tools/pytext"
)

// maxStrDigits is CPython's default sys.get_int_max_str_digits().
const maxStrDigits = 4300

// toBig returns an int or bool value as *big.Int (nil for other values).
func toBig(value any) *big.Int {
	if v, ok := value.(*big.Int); ok {
		return v
	}
	if number, ok := pytext.Int64(value); ok {
		return big.NewInt(number)
	}
	return nil
}

// intLike is isinstance(value, int): bool counts, as in Python.
func intLike(value any) bool { return pytext.IsIntOrBool(value) }

// small turns a big result back into int64 when it fits, like pyjson decodes integers.
func small(value *big.Int) any {
	if value.IsInt64() {
		return value.Int64()
	}
	return new(big.Int).Set(value)
}

// cmpInt compares two int or bool values (Python's < and == on ints).
func cmpInt(a, b any) int { return toBig(a).Cmp(toBig(b)) }

// bigFloat is float(int) with Python's OverflowError for ints beyond the float range.
func bigFloat(value *big.Int) (float64, error) {
	f, _ := new(big.Float).SetInt(value).Float64()
	if math.IsInf(f, 0) {
		return 0, crash("OverflowError", "int too large to convert to float")
	}
	return f, nil
}

// pyInt is int(value) for a JSON value; the result is int64 or *big.Int.
func pyInt(value any) (any, error) {
	switch v := value.(type) {
	case bool, int, int64:
		number, _ := pytext.Int64(v)
		return number, nil
	case *big.Int:
		return v, nil
	case float64:
		if math.IsNaN(v) {
			return nil, valueError("cannot convert float NaN to integer")
		}
		if math.IsInf(v, 0) {
			return nil, crash("OverflowError", "cannot convert float infinity to integer")
		}
		whole, _ := new(big.Float).SetFloat64(math.Trunc(v)).Int(nil)
		return small(whole), nil
	case string:
		return parseInt(v)
	}
	return nil, crash("TypeError",
		"int() argument must be a string, a bytes-like object or a real number, not '"+typeName(value)+"'")
}

// parseInt is int(text) with CPython's 4300-digit limit on decimal strings.
func parseInt(text string) (any, error) {
	number, err := pytext.ParseInt(text)
	if err != nil {
		return nil, valueError(err.Error())
	}
	if count := countDigits(text); count > maxStrDigits {
		return nil, valueError("Exceeds the limit (" + strconv.Itoa(maxStrDigits) +
			" digits) for integer string conversion: value has " + strconv.Itoa(count) +
			" digits; use sys.set_int_max_str_digits() to increase the limit")
	}
	return number, nil
}

// countDigits counts the digit characters of a valid int() literal (no sign, space, "_").
func countDigits(text string) int {
	count := 0
	for _, r := range pytext.Strip(text) {
		if r != '_' && r != '+' && r != '-' && !pytext.IsSpace(r) {
			count++
		}
	}
	return count
}

// pyFloat is float(value) for a JSON value.
func pyFloat(value any) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case *big.Int:
		return bigFloat(v)
	case bool, int, int64:
		number, _ := pytext.Int64(v)
		return float64(number), nil
	case string:
		number, err := pytext.ParseFloat(v)
		if err != nil {
			return 0, valueError(err.Error())
		}
		if strings.EqualFold(strings.TrimLeft(pytext.Strip(v), "+-"), "nan") {
			return math.NaN(), nil // pytext.ParseFloat gives 0 for a signed "-nan"
		}
		return number, nil
	}
	return 0, crash("TypeError", "float() argument must be a string or a real number, not '"+typeName(value)+"'")
}

// formatG is format(value, "g").
func formatG(value any) (string, error) {
	switch v := value.(type) {
	case float64:
		return pytext.FormatG(v, 6), nil
	case bool, int, int64, *big.Int:
		f, err := pyFloat(v)
		if err != nil {
			return "", err
		}
		return pytext.FormatG(f, 6), nil
	case string:
		return "", valueError("Unknown format code 'g' for object of type 'str'")
	}
	return "", crash("TypeError", "unsupported format string passed to "+typeName(value)+".__format__")
}

// floorDiv is a // b for int64 (Python rounds toward negative infinity).
func floorDiv(a, b int64) int64 {
	quotient := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		quotient--
	}
	return quotient
}
