// Package pycompat reproduces CPython 3.12 number and text rules the Python engine relied
// on (int() parsing, round(), str.strip(), str.casefold(), datetime isoformat and
// fromisoformat, utf-8-sig decoding, repr of str), so error texts and values the user sees
// stay identical. Every rule has a differential test against CPython (differential_test.go).
package pycompat

import (
	"strings"
	"unicode"
)

const lowerHex = "0123456789abcdef"

// Repr returns Python's repr() of a str: single quotes unless the text holds a single
// quote and no double quote; backslash escapes for the quote, '\\', \t, \n, \r; \xNN for
// other controls and DEL; non-printable characters as \xNN, \uNNNN or \UNNNNNNNN.
func Repr(text string) string {
	quote := '\''
	if strings.ContainsRune(text, '\'') && !strings.ContainsRune(text, '"') {
		quote = '"'
	}
	var out strings.Builder
	out.Grow(len(text) + 2)
	out.WriteRune(quote)
	for _, r := range text {
		switch {
		case r == quote || r == '\\':
			out.WriteByte('\\')
			out.WriteRune(r)
		case r == '\t':
			out.WriteString(`\t`)
		case r == '\n':
			out.WriteString(`\n`)
		case r == '\r':
			out.WriteString(`\r`)
		case r < ' ' || r == 0x7f:
			writeHex(&out, 'x', r, 2)
		case r < 0x7f:
			out.WriteRune(r)
		case IsPrintable(r):
			out.WriteRune(r)
		case r <= 0xff:
			writeHex(&out, 'x', r, 2)
		case r <= 0xffff:
			writeHex(&out, 'u', r, 4)
		default:
			writeHex(&out, 'U', r, 8)
		}
	}
	out.WriteRune(quote)
	return out.String()
}

// TruncatedRepr is CPython's "%.200R": the repr cut to its first 200 characters.
func TruncatedRepr(text string) string {
	repr := []rune(Repr(text))
	if len(repr) > 200 {
		repr = repr[:200]
	}
	return string(repr)
}

// IsPrintable is str.isprintable() for one character: everything except the Unicode
// categories Cc, Cf, Cs, Co, Cn, Zl, Zp and Zs, with the ASCII space printable.
func IsPrintable(r rune) bool {
	if r == ' ' {
		return true
	}
	return unicode.IsPrint(r)
}

func writeHex(out *strings.Builder, kind byte, r rune, width int) {
	out.WriteByte('\\')
	out.WriteByte(kind)
	for shift := (width - 1) * 4; shift >= 0; shift -= 4 {
		out.WriteByte(lowerHex[(r>>uint(shift))&0xf])
	}
}
