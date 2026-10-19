package pyv

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Casefold is Python's str.casefold() (full Unicode case folding, Unicode 15.0 as in
// CPython 3.12 and Go 1.24+). Invalid UTF-8 bytes become U+FFFD, as Flask decodes them.
// Stream A's pycompat.Casefold replaces this once it lands; the golden test is the same.
func Casefold(text string) string {
	ascii := true
	for index := 0; index < len(text); index++ {
		if text[index] >= utf8.RuneSelf {
			ascii = false
			break
		}
	}
	if ascii {
		return strings.ToLower(text)
	}
	var out strings.Builder
	out.Grow(len(text))
	for _, r := range text {
		out.WriteString(foldRune(r))
	}
	return out.String()
}

func foldRune(r rune) string {
	if special, found := foldSpecial[r]; found {
		return special
	}
	switch {
	case r >= 0x13A0 && r <= 0x13F5: // Cherokee capitals fold to themselves
		return string(r)
	case r >= 0x13F8 && r <= 0x13FD:
		return string(r - 8)
	case r >= 0xAB70 && r <= 0xABBF: // Cherokee small letters fold to the capitals
		return string(r - 0xAB70 + 0x13A0)
	}
	return string(unicode.ToLower(r))
}
