package xmldoc

import "unicode/utf8"

// reference checks the syntax of "&...;" at amp. On a syntax problem it returns the
// invalid-token position (bad >= 0) or eof=true when the input ends inside it.
func (s *scanner) reference(amp int) (end int, bad int, eof bool, partial bool) {
	p := amp + 1
	r, w, cls := s.at(p)
	switch {
	case cls == clsEOF:
		return 0, -1, true, false
	case cls == clsPartial:
		return 0, -1, false, true
	case cls == clsOK && r == '#':
		return s.charRefSyntax(p + 1)
	case cls != clsOK || !isNameStart(r):
		return 0, p, false, false
	}
	p += w
	nameEnd, ok, part := s.name(p)
	if part {
		return 0, -1, false, true
	}
	if !ok {
		return 0, nameEnd, false, false
	}
	r, _, cls = s.at(nameEnd)
	switch {
	case cls == clsEOF:
		return 0, -1, true, false
	case cls == clsOK && r == ';':
		return nameEnd + 1, -1, false, false
	}
	return 0, nameEnd, false, false
}

func (s *scanner) charRefSyntax(p int) (int, int, bool, bool) {
	hex := false
	if p < len(s.buf) && s.buf[p] == 'x' {
		hex = true
		p++
	}
	digits := 0
	for {
		if p >= len(s.buf) {
			return 0, -1, true, false
		}
		c := s.buf[p]
		switch {
		case c >= '0' && c <= '9', hex && (c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'):
			digits++
			p++
			continue
		case c == ';' && digits > 0:
			return p + 1, -1, false, false
		}
		if c >= 0x80 {
			if _, _, cls := s.at(p); cls == clsPartial {
				return 0, -1, false, true
			}
		}
		return 0, p, false, false
	}
}

// resolve returns the text of a syntactically valid reference amp..end, or the expat
// error (undefined entity or invalid character number).
func (s *scanner) resolveRef(amp, end int) (string, string) {
	body := s.buf[amp+1 : end-1]
	if body[0] == '#' {
		value, hex := 0, false
		digits := body[1:]
		if len(digits) > 0 && digits[0] == 'x' {
			hex, digits = true, digits[1:]
		}
		for _, c := range digits {
			d := 0
			switch {
			case c >= '0' && c <= '9':
				d = int(c - '0')
			case c >= 'a' && c <= 'f':
				d = int(c-'a') + 10
			default:
				d = int(c-'A') + 10
			}
			if hex {
				value = value<<4 | d
			} else {
				value = value*10 + d
			}
			if value >= 0x110000 {
				return "", errBadCharRef
			}
		}
		r := rune(value)
		if !(r == 9 || r == 10 || r == 13 || r >= 0x20 && r <= 0xd7ff || r >= 0xe000 && r <= 0xfffd ||
			r >= 0x10000 && r <= 0x10ffff) {
			return "", errBadCharRef
		}
		return string(utf8.AppendRune(nil, r)), ""
	}
	switch string(body) {
	case "lt":
		return "<", ""
	case "gt":
		return ">", ""
	case "amp":
		return "&", ""
	case "quot":
		return "\"", ""
	case "apos":
		return "'", ""
	}
	return "", errUndefined
}
