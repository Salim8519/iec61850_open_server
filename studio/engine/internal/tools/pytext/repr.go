// Package pytext reproduces the few CPython text and number rules the tools and gateway
// code depends on: str() and repr() of JSON values, format(x, "g"), int() and float() of
// text, str.strip() and truthiness. Stream A's pycompat will own the shared versions; these
// stay private to stream E until then (see streams/e-tools.md).
package pytext

import (
	"math/big"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/pyjson"
)

// Str returns Python's str(value) for a value decoded by pyjson.
func Str(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return Repr(value)
}

// Repr returns Python's repr(value) for a value decoded by pyjson.
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
		return FloatStr(v)
	case string:
		return ReprString(v)
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
			parts = append(parts, ReprString(key)+": "+Repr(item))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return "<unknown>"
}

// FloatStr is Python's str(float): repr digits, "nan", "inf" and "-inf".
func FloatStr(f float64) string {
	switch {
	case f != f:
		return "nan"
	case f > 1.7976931348623157e308:
		return "inf"
	case f < -1.7976931348623157e308:
		return "-inf"
	}
	return pyjson.Repr(f)
}

// ReprString is Python's repr(str): single quotes unless the text holds ' and no ",
// escapes for backslash, the quote, \t \n \r and every non-printable character.
func ReprString(text string) string {
	quote := byte('\'')
	if strings.ContainsRune(text, '\'') && !strings.ContainsRune(text, '"') {
		quote = '"'
	}
	out := []byte{quote}
	for index := 0; index < len(text); {
		r, size := utf8.DecodeRuneInString(text[index:])
		if r == utf8.RuneError && size == 1 {
			// Not reachable from JSON text (pyjson decodes to valid UTF-8); keep the byte visible.
			out = append(out, `\x`...)
			out = appendHex(out, uint32(text[index]), 2)
			index++
			continue
		}
		index += size
		switch {
		case r == '\\' || r == rune(quote):
			out = append(out, '\\', byte(r))
		case r == '\t':
			out = append(out, `\t`...)
		case r == '\n':
			out = append(out, `\n`...)
		case r == '\r':
			out = append(out, `\r`...)
		case r < 0x20 || r == 0x7f:
			out = append(out, `\x`...)
			out = appendHex(out, uint32(r), 2)
		case r < 0x7f || Printable(r):
			out = utf8.AppendRune(out, r)
		case r <= 0xff:
			out = append(out, `\x`...)
			out = appendHex(out, uint32(r), 2)
		case r <= 0xffff:
			out = append(out, `\u`...)
			out = appendHex(out, uint32(r), 4)
		default:
			out = append(out, `\U`...)
			out = appendHex(out, uint32(r), 8)
		}
	}
	return string(append(out, quote))
}

// Printable is str.isprintable for one character: not a control, format, surrogate,
// private-use, unassigned, line/paragraph separator or a space other than ' '.
func Printable(r rune) bool {
	if r == ' ' {
		return true
	}
	if unicode.In(r, unicode.Cc, unicode.Cf, unicode.Cs, unicode.Co, unicode.Zl, unicode.Zp, unicode.Zs) {
		return false
	}
	return unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S)
}

func appendHex(dst []byte, value uint32, width int) []byte {
	const digits = "0123456789abcdef"
	for shift := (width - 1) * 4; shift >= 0; shift -= 4 {
		dst = append(dst, digits[(value>>uint(shift))&0xf])
	}
	return dst
}
