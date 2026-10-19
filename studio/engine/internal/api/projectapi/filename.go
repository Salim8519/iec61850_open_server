package projectapi

import "strings"

// latinBase maps U+00C0..U+00FF to what NFKD + ASCII "ignore" keeps of them (Werkzeug's
// ASCII download-name fallback); characters without an ASCII base map to "".
var latinBase = []string{
	"A", "A", "A", "A", "A", "A", "", "C", "E", "E", "E", "E", "I", "I", "I", "I",
	"", "N", "O", "O", "O", "O", "O", "", "", "U", "U", "U", "U", "Y", "", "",
	"a", "a", "a", "a", "a", "a", "", "c", "e", "e", "e", "e", "i", "i", "i", "i",
	"", "n", "o", "o", "o", "o", "o", "", "", "u", "u", "u", "u", "y", "", "y",
}

// asciiFallback approximates unicodedata.normalize("NFKD", name).encode("ascii", "ignore")
// for Latin-1 letters; other non-ASCII characters are dropped (Python keeps the ASCII
// base of further compatibility decompositions, a documented corner of the port).
func asciiFallback(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r < 0x80:
			b.WriteRune(r)
		case r >= 0xc0 && r <= 0xff:
			b.WriteString(latinBase[r-0xc0])
		}
	}
	return b.String()
}
