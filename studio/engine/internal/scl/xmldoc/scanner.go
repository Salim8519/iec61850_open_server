package xmldoc

import (
	"strconv"
	"unicode"
	"unicode/utf8"
)

// Error is a parse failure with the exact text Python shows (str(error)).
type Error struct{ Text string }

func (e *Error) Error() string { return e.Text }

// expat error strings (XML_ErrorString) used by the scanner.
const (
	errSyntax       = "syntax error"
	errNoElements   = "no element found"
	errInvalid      = "not well-formed (invalid token)"
	errUnclosed     = "unclosed token"
	errPartial      = "partial character"
	errMismatch     = "mismatched tag"
	errDuplicate    = "duplicate attribute"
	errJunk         = "junk after document element"
	errUndefined    = "undefined entity"
	errBadCharRef   = "reference to invalid character number"
	errUnknownEnc   = "unknown encoding"
	errIncorrectEnc = "encoding specified in XML declaration is incorrect"
	errXMLDecl      = "XML declaration not well-formed"
	errMisplacedPI  = "XML or text declaration not at start of entity"
	errUnclosedCD   = "unclosed CDATA section"
	errUnbound      = "unbound prefix"
	errUndeclare    = "must not undeclare prefix"
	errReservedXML  = "reserved prefix (xml) must not be undeclared or bound to another namespace name"
	errReservedNS   = "reserved prefix (xmlns) must not be declared or undeclared"
	errReservedURI  = "prefix must not be bound to one of the reserved namespace names"
)

// char classes returned by scanner.at.
const (
	clsEOF = iota
	clsOK
	clsInvalid
	clsPartial
)

type scanner struct {
	buf     []byte
	start   int
	keep    bool
	oddTail bool
	doc     *Document
	ns      namespaces
}

// at decodes the character at p: rune, width and class (invalid = not an XML character
// or not decodable; partial = a multi-byte character cut by the end of the input).
func (s *scanner) at(p int) (rune, int, int) {
	if p >= len(s.buf) {
		return 0, 0, clsEOF
	}
	c := s.buf[p]
	if c < 0x80 {
		if c < 0x20 && c != '\t' && c != '\n' && c != '\r' {
			return rune(c), 1, clsInvalid
		}
		return rune(c), 1, clsOK
	}
	r, w := utf8.DecodeRune(s.buf[p:])
	if r == utf8.RuneError && w <= 1 {
		if !utf8.FullRune(s.buf[p:]) && validPrefix(s.buf[p:]) {
			return 0, len(s.buf) - p, clsPartial
		}
		return 0, 1, clsInvalid
	}
	if r == 0xfffe || r == 0xffff {
		return r, w, clsInvalid
	}
	return r, w, clsOK
}

// validPrefix reports whether b (shorter than its announced length) could still become a
// valid UTF-8 character; expat then says "partial character" at the end of the input.
func validPrefix(b []byte) bool {
	n := 0
	switch c := b[0]; {
	case c >= 0xc2 && c <= 0xdf:
		n = 2
	case c >= 0xe0 && c <= 0xef:
		n = 3
	case c >= 0xf0 && c <= 0xf4:
		n = 4
	default:
		return false
	}
	if len(b) >= n {
		return false
	}
	for i := 1; i < len(b); i++ {
		if b[i]&0xc0 != 0x80 {
			return false
		}
	}
	if len(b) >= 2 {
		switch b[0] {
		case 0xe0:
			return b[1] >= 0xa0
		case 0xed:
			return b[1] < 0xa0
		case 0xf0:
			return b[1] >= 0x90
		case 0xf4:
			return b[1] < 0x90
		}
	}
	return true
}

// fail builds the error text with expat's position of p: lines end at LF, CR or CRLF and
// columns count characters from 0.
func (s *scanner) fail(message string, p int) error {
	line, column := 1, 0
	for i := 0; i < p && i < len(s.buf); i++ {
		switch c := s.buf[i]; {
		case c == '\n':
			line, column = line+1, 0
		case c == '\r':
			line, column = line+1, 0
			if i+1 < p && s.buf[i+1] == '\n' {
				i++
			}
		case c&0xc0 != 0x80:
			column++
		}
	}
	return &Error{Text: message + ": line " + strconv.Itoa(line) + ", column " + strconv.Itoa(column)}
}

func isSpace(c byte) bool { return c == ' ' || c == '\t' || c == '\n' || c == '\r' }

// isNameStart and isNameChar follow expat's name classes: ASCII letters, '_' (':' is
// handled by the namespace-aware scanner), BMP letters; supplementary characters never.
func isNameStart(r rune) bool {
	if r < 0x80 {
		return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_'
	}
	if r >= 0x10000 || r >= 0x3400 && r <= 0x4dbf || r >= 0x9fa6 && r <= 0x9fff {
		return false // expat's name tables predate CJK extension A and later ideographs
	}
	return unicode.IsLetter(r) || unicode.Is(unicode.Nl, r)
}

func isNameChar(r rune) bool {
	if r < 0x80 {
		return isNameStart(r) || r >= '0' && r <= '9' || r == '-' || r == '.'
	}
	if r >= 0x10000 {
		return false
	}
	return isNameStart(r) || r == 0xb7 || unicode.In(r, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Lm)
}

// name scans an XML name with at most one colon starting at p (the first character was
// already checked as a name start). It returns the end, or a failing position with ok=false.
func (s *scanner) name(p int) (end int, ok bool, partial bool) {
	colon := false
	for {
		r, w, cls := s.at(p)
		switch {
		case cls == clsPartial:
			return p, false, true
		case cls == clsOK && r == ':':
			next, _, nextCls := s.at(p + 1)
			if colon || nextCls != clsOK || !isNameStart(next) {
				if nextCls == clsEOF {
					return p + 1, true, false
				}
				if colon {
					return p, false, false
				}
				return p + 1, false, false
			}
			colon = true
			p++
		case cls == clsOK && isNameChar(r):
			p += w
		default:
			return p, true, false
		}
	}
}
