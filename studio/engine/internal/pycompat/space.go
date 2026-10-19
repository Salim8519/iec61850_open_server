package pycompat

import (
	"strings"
	"unicode"
)

// IsSpace is str.isspace() for one character: Unicode category Zs or bidirectional class
// WS, B or S. Unlike Go's unicode.IsSpace it includes the separators U+001C..U+001F.
func IsSpace(r rune) bool {
	if r >= 0x1c && r <= 0x1f {
		return true
	}
	return unicode.IsSpace(r)
}

// IsASCIISpace is CPython's Py_ISSPACE: space, \t, \n, \v, \f and \r only.
func IsASCIISpace(b byte) bool {
	return b == ' ' || (b >= '\t' && b <= '\r')
}

// Strip is str.strip() without arguments.
func Strip(text string) string { return strings.TrimFunc(text, IsSpace) }

// LStrip is str.lstrip() without arguments.
func LStrip(text string) string { return strings.TrimLeftFunc(text, IsSpace) }

// RStrip is str.rstrip() without arguments.
func RStrip(text string) string { return strings.TrimRightFunc(text, IsSpace) }

// Split is str.split() without arguments: runs of whitespace separate fields and leading
// or trailing whitespace gives no empty field.
func Split(text string) []string { return strings.FieldsFunc(text, IsSpace) }
