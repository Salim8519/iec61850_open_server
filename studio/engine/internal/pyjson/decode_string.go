package pyjson

import (
	"unicode/utf16"
	"unicode/utf8"
)

// string reads a JSON string at p.pos (on the opening quote) like json.loads in strict mode.
// A lone surrogate escape, which Python keeps, becomes U+FFFD here (Go strings are UTF-8).
func (p *parser) string() (string, error) {
	p.pos++
	out := make([]byte, 0, 16)
	for {
		if p.pos >= len(p.data) {
			return "", p.fail("Unterminated string starting")
		}
		c := p.data[p.pos]
		switch {
		case c == '"':
			p.pos++
			return string(out), nil
		case c < 0x20:
			return "", p.fail("Invalid control character")
		case c != '\\':
			out = append(out, c)
			p.pos++
			continue
		}
		p.pos++
		if p.pos >= len(p.data) {
			return "", p.fail("Unterminated string starting")
		}
		escape := p.data[p.pos]
		p.pos++
		switch escape {
		case '"', '\\', '/':
			out = append(out, escape)
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'u':
			r, err := p.unicodeEscape()
			if err != nil {
				return "", err
			}
			out = utf8.AppendRune(out, r)
		default:
			return "", p.fail("Invalid \\escape")
		}
	}
}

func (p *parser) unicodeEscape() (rune, error) {
	first, ok := p.hex4()
	if !ok {
		return 0, p.fail("Invalid \\uXXXX escape")
	}
	if utf16.IsSurrogate(first) && first < 0xdc00 && p.pos+1 < len(p.data) &&
		p.data[p.pos] == '\\' && p.data[p.pos+1] == 'u' {
		mark := p.pos
		p.pos += 2
		if second, ok := p.hex4(); ok && second >= 0xdc00 && second <= 0xdfff {
			return utf16.DecodeRune(first, second), nil
		}
		p.pos = mark
	}
	if utf16.IsSurrogate(first) {
		return utf8.RuneError, nil
	}
	return first, nil
}

func (p *parser) hex4() (rune, bool) {
	if len(p.data)-p.pos < 4 {
		return 0, false
	}
	var value rune
	for _, c := range p.data[p.pos : p.pos+4] {
		value <<= 4
		switch {
		case c >= '0' && c <= '9':
			value |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			value |= rune(c-'a') + 10
		case c >= 'A' && c <= 'F':
			value |= rune(c-'A') + 10
		default:
			return 0, false
		}
	}
	p.pos += 4
	return value, true
}
