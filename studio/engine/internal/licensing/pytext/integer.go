package pytext

import (
	"errors"
	"math/big"
	"strings"
	"unicode"
)

// maxDigits is CPython's default int max_str_digits limit.
const maxDigits = 4300

// ErrInvalidInt stands for the ValueError int(text) raises.
var ErrInvalidInt = errors.New("invalid literal for int() with base 10")

// DigitValue returns the decimal value of a Unicode decimal digit (category Nd), which
// Python's int() and the \d class of str regular expressions accept like ASCII digits.
// Unicode keeps every Nd run as complete, consecutive 0-9 sequences.
func DigitValue(r rune) (int, bool) {
	if !unicode.IsDigit(r) {
		return 0, false
	}
	steps := 0
	for unicode.IsDigit(r - rune(steps) - 1) {
		steps++
	}
	return steps % 10, true
}

// intSpace is the whitespace int() skips: ASCII " \t\n\v\f\r" (Py_ISSPACE) and non-ASCII
// str whitespace, which CPython first turns into a space; \x1c-\x1f are not skipped.
func intSpace(r rune) bool {
	if r < 0x80 {
		return r == ' ' || (r >= '\t' && r <= '\r')
	}
	return IsSpace(r)
}

// ParseInt returns int(text) for a str argument: surrounding whitespace, one optional
// sign, decimal digits of any script and single underscores between digits.
func ParseInt(text string) (*big.Int, error) {
	runes := []rune(strings.TrimFunc(text, intSpace))
	negative := false
	if len(runes) > 0 && (runes[0] == '+' || runes[0] == '-') {
		negative = runes[0] == '-'
		runes = runes[1:]
	}
	value, digits, previousDigit := new(big.Int), 0, false
	ten := big.NewInt(10)
	for _, r := range runes {
		if r == '_' {
			if !previousDigit {
				return nil, ErrInvalidInt
			}
			previousDigit = false
			continue
		}
		digit, ok := DigitValue(r)
		if !ok {
			return nil, ErrInvalidInt
		}
		value.Mul(value, ten).Add(value, big.NewInt(int64(digit)))
		digits++
		previousDigit = true
	}
	if digits == 0 || !previousDigit || digits > maxDigits {
		return nil, ErrInvalidInt
	}
	if negative {
		value.Neg(value)
	}
	return value, nil
}
