package pyval

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
)

// ConvError is the ValueError/TypeError text float() or int() raises.
type ConvError struct {
	Message string
	Type    bool // TypeError (HTTP 500 in Flask) rather than ValueError
}

func (e *ConvError) Error() string { return e.Message }

func isSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x1f, 0x85, 0xa0, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

func digitsWithUnderscores(text string) (string, bool) {
	if strings.HasPrefix(text, "_") || strings.HasSuffix(text, "_") || strings.Contains(text, "__") {
		return "", false
	}
	return strings.ReplaceAll(text, "_", ""), true
}

// ToFloat is float(value) for JSON values.
func ToFloat(value any) (float64, error) {
	switch typed := value.(type) {
	case string:
		text := strings.TrimFunc(typed, isSpace)
		lower := strings.ToLower(strings.TrimLeft(text, "+-"))
		if lower == "inf" || lower == "infinity" || lower == "nan" {
			parsed, _ := strconv.ParseFloat(strings.Replace(text, "inity", "", 1), 64)
			if lower == "nan" {
				parsed = math.NaN()
			}
			return parsed, nil
		}
		clean, ok := digitsWithUnderscores(text)
		if ok && clean != "" && !strings.ContainsAny(clean, "xXpP") {
			if parsed, err := strconv.ParseFloat(clean, 64); err == nil || isRange(err) {
				return parsed, nil
			}
		}
		return 0, &ConvError{Message: fmt.Sprintf("could not convert string to float: %s", inner(typed))}
	case nil, []any:
		return 0, &ConvError{Type: true, Message: fmt.Sprintf("float() argument must be a string or a real number, not '%s'", TypeName(value))}
	}
	if f, ok := Float(value); ok {
		return f, nil
	}
	return 0, &ConvError{Type: true, Message: fmt.Sprintf("float() argument must be a string or a real number, not '%s'", TypeName(value))}
}

func isRange(err error) bool {
	numErr, ok := err.(*strconv.NumError)
	return ok && numErr.Err == strconv.ErrRange
}

// ToInt is int(value): floats truncate toward zero, strings parse base 10.
func ToInt(value any) (any, error) {
	switch typed := value.(type) {
	case bool:
		if typed {
			return int64(1), nil
		}
		return int64(0), nil
	case int64, *big.Int:
		return typed, nil
	case int:
		return int64(typed), nil
	case float64:
		if math.IsNaN(typed) {
			return nil, &ConvError{Message: "cannot convert float NaN to integer"}
		}
		if math.IsInf(typed, 0) {
			return nil, &ConvError{Message: "cannot convert float infinity to integer"}
		}
		truncated := math.Trunc(typed)
		if math.Abs(truncated) < 9.2e18 {
			return int64(truncated), nil
		}
		result, _ := new(big.Float).SetFloat64(truncated).Int(nil)
		return result, nil
	case string:
		text := strings.TrimFunc(typed, isSpace)
		clean, ok := digitsWithUnderscores(strings.TrimLeft(text, "+-"))
		if ok && clean != "" && strings.Trim(clean, "0123456789") == "" && len(text)-len(strings.TrimLeft(text, "+-")) <= 1 {
			sign := text[:len(text)-len(strings.TrimLeft(text, "+-"))]
			parsed, _ := new(big.Int).SetString(sign+clean, 10)
			if parsed.IsInt64() {
				return parsed.Int64(), nil
			}
			return parsed, nil
		}
		return nil, &ConvError{Message: fmt.Sprintf("invalid literal for int() with base 10: %s", inner(typed))}
	}
	return nil, &ConvError{Type: true, Message: fmt.Sprintf("int() argument must be a string, a bytes-like object or a real number, not '%s'", TypeName(value))}
}
