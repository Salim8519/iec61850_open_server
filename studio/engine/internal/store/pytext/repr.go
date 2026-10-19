// Package pytext writes Python's text forms of values that leak into stored files and
// messages: str() and repr() of JSON values, str(Path) and str(OSError) on Windows.
package pytext

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/pyjson"
)

// Str returns Python's str(value) for a decoded JSON value: text unchanged, None,
// True/False, int digits, float repr (nan, inf), and repr() of lists and dicts.
func Str(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return Repr(value)
}

// Repr returns Python's repr(value) for a decoded JSON value.
func Repr(value any) string {
	switch v := value.(type) {
	case nil:
		return "None"
	case bool:
		if v {
			return "True"
		}
		return "False"
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case *big.Int:
		return v.String()
	case float64:
		return floatRepr(v)
	case string:
		return StrRepr(v)
	case []any:
		parts := make([]string, len(v))
		for index, item := range v {
			parts[index] = Repr(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *pyjson.Object:
		parts := make([]string, 0, v.Len())
		for _, key := range v.Keys() {
			item, _ := v.Get(key)
			parts = append(parts, StrRepr(key)+": "+Repr(item))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return fmt.Sprint(value)
}

func floatRepr(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	return pyjson.Repr(f)
}

// StrRepr returns Python's repr of a str: single quotes unless the text holds ' and no ",
// backslash escapes for \\ \t \n \r and the quote, \xNN / \uNNNN / \UNNNNNNNN for
// characters that are not printable. Invalid UTF-8 bytes stand for U+FFFD.
func StrRepr(text string) string {
	quote := byte('\'')
	if strings.Contains(text, "'") && !strings.Contains(text, "\"") {
		quote = '"'
	}
	out := []byte{quote}
	for _, r := range text {
		switch {
		case r == rune(quote) || r == '\\':
			out = append(out, '\\', byte(r))
		case r == '\t':
			out = append(out, '\\', 't')
		case r == '\n':
			out = append(out, '\\', 'n')
		case r == '\r':
			out = append(out, '\\', 'r')
		case r < 0x20 || r == 0x7f:
			out = fmt.Appendf(out, "\\x%02x", r)
		case r < 0x7f || printable(r):
			out = utf8.AppendRune(out, r)
		case r <= 0xff:
			out = fmt.Appendf(out, "\\x%02x", r)
		case r <= 0xffff:
			out = fmt.Appendf(out, "\\u%04x", r)
		default:
			out = fmt.Appendf(out, "\\U%08x", r)
		}
	}
	return string(append(out, quote))
}

// printable approximates str.isprintable: letters, marks, numbers, punctuation and
// symbols; other separators and control/format/private/unassigned characters are not.
func printable(r rune) bool {
	return unicode.IsPrint(r) && r != 0xad
}

// Slice returns text[:limit] counted in code points, like a Python str slice.
func Slice(text string, limit int) string {
	count := 0
	for index := range text {
		if count == limit {
			return text[:index]
		}
		count++
	}
	return text
}

// Len returns len(text) in code points.
func Len(text string) int { return utf8.RuneCountInString(text) }
