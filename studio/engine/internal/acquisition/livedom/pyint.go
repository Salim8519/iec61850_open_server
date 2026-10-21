package livedom

import (
	"math"
	"math/big"
	"strconv"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// maxIntDigits is CPython's default sys.get_int_max_str_digits().
const maxIntDigits = 4300

// PyError is a Python exception the Go port raises at the same place: Class is the Python
// class name (ValueError, TypeError, OverflowError, AttributeError) and Message its str().
type PyError struct {
	Class   string
	Message string
}

func (e *PyError) Error() string { return e.Message }

// Text is the "Class: message" form the golden generators record.
func (e *PyError) Text() string { return e.Class + ": " + e.Message }

// ParseIntText is Python's int(text) for a str: surrounding Unicode whitespace, an optional
// sign, decimal digits of any script and single underscores between digits. It fails with
// Python's ValueError text (the repr of text is cut at 200 characters like CPython's %.200R).
func ParseIntText(text string) (*big.Int, error) {
	ascii := make([]byte, 0, len(text))
	for _, r := range text {
		switch {
		case r < utf8.RuneSelf:
			ascii = append(ascii, byte(r))
		case isPySpace(r):
			ascii = append(ascii, ' ')
		default:
			if digit, ok := decimalValue(r); ok {
				ascii = append(ascii, byte('0'+digit))
			} else {
				ascii = append(ascii, '?')
			}
		}
	}
	digits, ok := scanDecimal(ascii)
	if !ok {
		return nil, &PyError{"ValueError", "invalid literal for int() with base 10: " + clipRunes(pyv.Repr(text), 200)}
	}
	count := len(digits)
	if digits[0] == '-' || digits[0] == '+' {
		count--
	}
	if count > maxIntDigits {
		return nil, &PyError{"ValueError", "Exceeds the limit (4300 digits) for integer string conversion: " +
			"value has " + strconv.Itoa(count) + " digits; use sys.set_int_max_str_digits() to increase the limit"}
	}
	value, _ := new(big.Int).SetString(string(digits), 10)
	return value, nil
}

// scanDecimal validates CPython's ASCII form and returns the sign and digits without
// underscores, or false when the literal is invalid.
func scanDecimal(text []byte) ([]byte, bool) {
	start, end := 0, len(text)
	for start < end && isASCIISpace(text[start]) {
		start++
	}
	for end > start && isASCIISpace(text[end-1]) {
		end--
	}
	var out []byte
	if start < end && (text[start] == '+' || text[start] == '-') {
		out = append(out, text[start])
		start++
	}
	if start == end || !isASCIIDigit(text[start]) {
		return nil, false
	}
	for index := start; index < end; index++ {
		switch char := text[index]; {
		case isASCIIDigit(char):
			out = append(out, char)
		case char == '_' && index+1 < end && isASCIIDigit(text[index+1]):
		default:
			return nil, false
		}
	}
	return out, true
}

func isASCIISpace(char byte) bool { return char == ' ' || (char >= '\t' && char <= '\r') }

func isASCIIDigit(char byte) bool { return char >= '0' && char <= '9' }

// isPySpace is Py_UNICODE_ISSPACE for code points from U+0080 on.
func isPySpace(r rune) bool {
	switch r {
	case 0x85, 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

// decimalValue is Py_UNICODE_TODECIMAL: Unicode decimal digits (category Nd) come in runs
// of ten consecutive code points from zero to nine.
func decimalValue(r rune) (int, bool) {
	if !unicode.Is(unicode.Nd, r) {
		return 0, false
	}
	start := r
	for start > 0 && unicode.Is(unicode.Nd, start-1) {
		start--
	}
	return int(r-start) % 10, true
}

// clipRunes keeps the first limit code points of text.
func clipRunes(text string, limit int) string {
	count := 0
	for index := range text {
		if count == limit {
			return text[:index]
		}
		count++
	}
	return text
}

// PyInt is Python's int(value) for pyjson values: integers stay, floats truncate toward
// zero, bool is 0/1, strings parse like ParseIntText. Python ints outside int64 are not
// representable here and fail with an OverflowError (Python would keep the big value).
func PyInt(value any) (int64, error) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	case float64:
		switch {
		case math.IsNaN(v):
			return 0, &PyError{"ValueError", "cannot convert float NaN to integer"}
		case math.IsInf(v, 0):
			return 0, &PyError{"OverflowError", "cannot convert float infinity to integer"}
		case math.Abs(v) >= 9.223372036854775807e18:
			return 0, outOfRange()
		}
		return int64(v), nil
	case string:
		parsed, err := ParseIntText(v)
		if err != nil {
			return 0, err
		}
		if !parsed.IsInt64() {
			return 0, outOfRange()
		}
		return parsed.Int64(), nil
	}
	if number, ok := pyv.Int64(value); ok {
		return number, nil
	}
	if pyv.IsInt(value) {
		return 0, outOfRange()
	}
	return 0, &PyError{"TypeError", "int() argument must be a string, a bytes-like object or a real number, not '" +
		TypeName(value) + "'"}
}

func outOfRange() error {
	return &PyError{"OverflowError", "Python int too large to convert to C int64"}
}

// TypeName is type(value).__name__ for pyjson values.
func TypeName(value any) string {
	switch value.(type) {
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
	return "object"
}
