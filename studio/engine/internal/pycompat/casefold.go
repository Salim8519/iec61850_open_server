package pycompat

import (
	"strings"
	"unicode"
)

type foldRange struct {
	lo, hi rune
	delta  rune
}

// Casefold is str.casefold(): Unicode full case folding (CaseFolding.txt C and F), so
// "Straße" and "STRASSE" fold to the same text. Characters without a folding entry use
// Python's full lowercase, which equals Go's simple lowercase except for the table.
func Casefold(text string) string {
	var out strings.Builder
	out.Grow(len(text))
	for _, r := range text {
		out.WriteString(CasefoldRune(r))
	}
	return out.String()
}

// CasefoldRune is the case folding of one character (one or more characters).
func CasefoldRune(r rune) string {
	if r < 0x80 {
		if r >= 'A' && r <= 'Z' {
			r += 'a' - 'A'
		}
		return string(r)
	}
	if expansion, found := foldExpansions[r]; found {
		return expansion
	}
	for _, span := range foldRanges {
		if r < span.lo {
			break
		}
		if r <= span.hi {
			return string(r + span.delta)
		}
	}
	return string(unicode.ToLower(r))
}
