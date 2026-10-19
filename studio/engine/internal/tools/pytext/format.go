package pytext

import (
	"math"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// FormatG is format(f, ".<precision>g"): precision significant digits, trailing zeros
// removed, scientific form when the exponent is below -4 or at least the precision, and
// "nan", "inf", "-inf" for the special values. Go's 'g' verb follows the same C rules.
func FormatG(f float64, precision int) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	if precision == 0 {
		precision = 1
	}
	return strconv.FormatFloat(f, 'g', precision, 64)
}

// IsSpace is Python's str.isspace for one character (str.strip and str.split use it).
func IsSpace(r rune) bool {
	return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f)
}

// Strip is str.strip() without arguments.
func Strip(text string) string {
	return strings.TrimFunc(text, IsSpace)
}

// numberSpace is the whitespace int() and float() ignore around a number: ASCII
// " \t\n\v\f\r" and every non-ASCII Unicode space (\x1c-\x1f are not stripped there).
func numberSpace(r rune) bool {
	if r < 0x80 {
		return r == ' ' || (r >= '\t' && r <= '\r')
	}
	return IsSpace(r)
}

// Len is Python's len(str): the number of code points.
func Len(text string) int { return utf8.RuneCountInString(text) }

// Head is text[:limit] in code points.
func Head(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	count := 0
	for index := range text {
		if count == limit {
			return text[:index]
		}
		count++
	}
	return text
}

// Truthy is Python's bool(value) for a value decoded by pyjson.
func Truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []any:
		return len(v) > 0
	case interface{ Len() int }:
		return v.Len() > 0
	case interface{ Sign() int }:
		return v.Sign() != 0
	}
	return true
}

// Capitalize is str.capitalize for the ASCII setting names it is used on.
func Capitalize(text string) string {
	if text == "" {
		return text
	}
	return strings.ToUpper(text[:1]) + strings.ToLower(text[1:])
}
