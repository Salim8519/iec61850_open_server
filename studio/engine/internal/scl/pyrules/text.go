// Package pyrules holds the few CPython text and number rules the SCL and project code
// depend on (str.strip, str.isspace, repr). It is a pure leaf package; at integration it
// folds into internal/pycompat (stream A) without changing behaviour.
package pyrules

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// IsSpace reports whether r is whitespace for Python's str.isspace and str.strip().
func IsSpace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x1f, 0x85, 0xa0, 0x1680,
		0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

// Strip is Python's str.strip() without arguments.
func Strip(s string) string {
	return strings.TrimFunc(s, IsSpace)
}

// HasSpace reports whether any character of s is Python whitespace.
func HasSpace(s string) bool {
	return strings.IndexFunc(s, IsSpace) >= 0
}

// Lower is Python's str.lower() for the texts SCL compares (full Unicode lower case).
func Lower(s string) string {
	return strings.ToLower(s)
}

// Repr returns Python's repr() of a str.
func Repr(s string) string {
	quote := byte('\'')
	if strings.ContainsRune(s, '\'') && !strings.ContainsRune(s, '"') {
		quote = '"'
	}
	var b strings.Builder
	b.Grow(len(s) + 2)
	b.WriteByte(quote)
	for _, r := range s {
		switch {
		case r == rune(quote) || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '\t':
			b.WriteString(`\t`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r < 0x20 || r == 0x7f:
			writeHex(&b, `\x`, r, 2)
		case r < 0x7f:
			b.WriteRune(r)
		case printable(r):
			b.WriteRune(r)
		case r <= 0xff:
			writeHex(&b, `\x`, r, 2)
		case r <= 0xffff:
			writeHex(&b, `\u`, r, 4)
		default:
			writeHex(&b, `\U`, r, 8)
		}
	}
	b.WriteByte(quote)
	return b.String()
}

// ReprOrNone is repr() of an optional str: None when absent.
func ReprOrNone(s string, present bool) string {
	if !present {
		return "None"
	}
	return Repr(s)
}

// printable follows str.isprintable: everything except Cc, Cf, Cs, Co, Cn, Zl, Zp, Zs.
func printable(r rune) bool {
	if r == utf8.RuneError {
		return true
	}
	if unicode.In(r, unicode.Cc, unicode.Cf, unicode.Cs, unicode.Co, unicode.Zl, unicode.Zp, unicode.Zs) {
		return false
	}
	return unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S)
}

func writeHex(b *strings.Builder, prefix string, r rune, width int) {
	const digits = "0123456789abcdef"
	b.WriteString(prefix)
	for shift := (width - 1) * 4; shift >= 0; shift -= 4 {
		b.WriteByte(digits[(r>>uint(shift))&0xf])
	}
}
