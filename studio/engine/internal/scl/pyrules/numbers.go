package pyrules

import (
	"errors"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

// ErrInvalid is returned when Python's int() or float() would raise ValueError.
var ErrInvalid = errors.New("invalid literal")

// IntMessage is Python's ValueError text for int(text) with base 10.
func IntMessage(text string) string {
	return "invalid literal for int() with base 10: " + Repr(text)
}

// FloatMessage is Python's ValueError text for float(text).
func FloatMessage(text string) string {
	return "could not convert string to float: " + Repr(text)
}

// Int parses text like Python's int(text, base) for base 10 or 16. Values outside int64
// are returned as *big.Int through IntBig; Int itself reports them as ErrInvalid-free
// overflow through ok=false only when they do not fit.
func Int(text string, base int) (int64, error) {
	value, err := IntBig(text, base)
	if err != nil {
		return 0, err
	}
	if !value.IsInt64() {
		return 0, errOverflow
	}
	return value.Int64(), nil
}

var errOverflow = errors.New("integer outside int64")

// IsOverflow reports whether err means the text was a valid but too large integer.
func IsOverflow(err error) bool { return errors.Is(err, errOverflow) }

// IntBig parses text like Python's int(text, base) (base 10 or 16) without size limit.
func IntBig(text string, base int) (*big.Int, error) {
	s := Strip(text)
	negative := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		negative = s[0] == '-'
		s = s[1:]
	}
	if base == 16 && len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
		if strings.HasPrefix(s, "_") {
			s = s[1:]
		}
	}
	digits, ok := cleanDigits(s, base)
	if !ok {
		return nil, ErrInvalid
	}
	value, ok := new(big.Int).SetString(digits, base)
	if !ok {
		return nil, ErrInvalid
	}
	if negative {
		value.Neg(value)
	}
	return value, nil
}

// cleanDigits removes single underscores between digits and maps Unicode decimal digits.
func cleanDigits(s string, base int) (string, bool) {
	if s == "" {
		return "", false
	}
	var b strings.Builder
	previousUnderscore := true
	for _, r := range s {
		if r == '_' {
			if previousUnderscore {
				return "", false
			}
			previousUnderscore = true
			continue
		}
		previousUnderscore = false
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case base == 16 && (r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F'):
			b.WriteRune(r)
		case r > 0x7f && unicode.Is(unicode.Nd, r):
			b.WriteRune('0' + decimalValue(r))
		default:
			return "", false
		}
	}
	if previousUnderscore {
		return "", false
	}
	return b.String(), true
}

// decimalValue returns the digit value of a Unicode Nd character: Nd characters come in
// contiguous runs made of complete 0-9 sets, so the value is the offset in the run mod 10.
func decimalValue(r rune) rune {
	start := r
	for unicode.Is(unicode.Nd, start-1) {
		start--
	}
	return (r - start) % 10
}

// Float parses text like Python's float(text). Infinite results (1e309) are returned
// without error, as Python does.
func Float(text string) (float64, error) {
	s := Strip(text)
	body := s
	if body != "" && (body[0] == '+' || body[0] == '-') {
		body = body[1:]
	}
	switch strings.ToLower(body) {
	case "inf", "infinity":
		if s[0] == '-' {
			return math.Inf(-1), nil
		}
		return math.Inf(1), nil
	case "nan":
		return math.NaN(), nil
	}
	cleaned, ok := floatDigits(s)
	if !ok {
		return 0, ErrInvalid
	}
	value, err := strconv.ParseFloat(cleaned, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, ErrInvalid
	}
	return value, nil
}

// floatDigits validates a decimal float literal (digits, one dot, optional exponent,
// underscores only between digits) and returns it without underscores.
func floatDigits(s string) (string, bool) {
	var b strings.Builder
	state, digitsSeen, expDigits := 0, false, false
	previous := rune(0)
	for index, r := range s {
		if r > 0x7f && unicode.Is(unicode.Nd, r) {
			r = '0' + decimalValue(r)
		}
		switch {
		case r == '+' || r == '-':
			if !(index == 0 || previous == 'e' || previous == 'E') {
				return "", false
			}
		case r >= '0' && r <= '9':
			if state == 2 {
				expDigits = true
			} else {
				digitsSeen = true
			}
		case r == '_':
			if previous < '0' || previous > '9' {
				return "", false
			}
			previous = r
			continue
		case r == '.':
			if state != 0 || previous == '_' {
				return "", false
			}
			state = 1
		case r == 'e' || r == 'E':
			if state == 2 || !digitsSeen || previous == '_' {
				return "", false
			}
			state = 2
		default:
			return "", false
		}
		if previous == '_' && (r < '0' || r > '9') {
			return "", false
		}
		previous = r
		b.WriteRune(r)
	}
	if previous == '_' || !digitsSeen || (state == 2 && !expDigits) {
		return "", false
	}
	return b.String(), true
}

// Round is Python 3's round(x) to an integer: half to even.
func Round(x float64) float64 {
	return math.RoundToEven(x)
}
