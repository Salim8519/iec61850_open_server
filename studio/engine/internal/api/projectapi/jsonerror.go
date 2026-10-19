package projectapi

import (
	"regexp"
	"strconv"
	"strings"
)

// pythonJSONError returns str(json.JSONDecodeError) that json.loads(text) raises, or ""
// when text is valid JSON. It follows CPython 3.12's json scanner (C accelerated) for the
// messages and positions; positions count characters.
func pythonJSONError(text string) string {
	runes := []rune(text)
	if len(runes) > 0 && runes[0] == 0xfeff {
		return jsonMessage("Unexpected UTF-8 BOM (decode using utf-8-sig)", runes, 0)
	}
	p := &jsonScan{s: runes}
	end, failure := p.value(skipWS(runes, 0))
	if failure != "" {
		return failure
	}
	if end = skipWS(runes, end); end != len(runes) {
		return jsonMessage("Extra data", runes, end)
	}
	return ""
}

func jsonMessage(message string, s []rune, pos int) string {
	line := 1
	last := -1
	for i := 0; i < pos && i < len(s); i++ {
		if s[i] == '\n' {
			line++
			last = i
		}
	}
	return message + ": line " + strconv.Itoa(line) + " column " + strconv.Itoa(pos-last) + " (char " + strconv.Itoa(pos) + ")"
}

func skipWS(s []rune, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i
}

type jsonScan struct{ s []rune }

var jsonNumber = regexp.MustCompile(`^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?(?:[eE][-+]?[0-9]+)?`)

func (p *jsonScan) at(i int) rune {
	if i < len(p.s) {
		return p.s[i]
	}
	return 0
}

// value is scan_once: the end of the value or the error text.
func (p *jsonScan) value(i int) (int, string) {
	s := p.s
	switch c := p.at(i); {
	case i >= len(s):
		return 0, jsonMessage("Expecting value", s, i)
	case c == '"':
		return p.str(i + 1)
	case c == '{':
		return p.object(i + 1)
	case c == '[':
		return p.array(i + 1)
	}
	rest := string(s[i:])
	for _, word := range []string{"null", "true", "false"} {
		if strings.HasPrefix(rest, word) {
			return i + len(word), ""
		}
	}
	if m := jsonNumber.FindString(rest); m != "" {
		return i + len(m), ""
	}
	for _, word := range []string{"NaN", "Infinity", "-Infinity"} {
		if strings.HasPrefix(rest, word) {
			return i + len(word), ""
		}
	}
	return 0, jsonMessage("Expecting value", s, i)
}

func (p *jsonScan) str(i int) (int, string) {
	begin := i - 1
	for {
		if i >= len(p.s) {
			return 0, jsonMessage("Unterminated string starting at", p.s, begin)
		}
		c := p.s[i]
		switch {
		case c == '"':
			return i + 1, ""
		case c == '\\':
			esc := p.at(i + 1)
			if i+1 >= len(p.s) {
				return 0, jsonMessage("Unterminated string starting at", p.s, begin)
			}
			if esc == 'u' {
				for k := 2; k < 6; k++ {
					if !strings.ContainsRune("0123456789abcdefABCDEF", p.at(i+k)) || i+k >= len(p.s) {
						return 0, jsonMessage("Invalid \\uXXXX escape", p.s, i+1)
					}
				}
				i += 6
				continue
			}
			if !strings.ContainsRune(`"\/bfnrt`, esc) {
				return 0, jsonMessage("Invalid \\escape", p.s, i)
			}
			i += 2
		case c < 0x20:
			return 0, jsonMessage("Invalid control character at", p.s, i)
		default:
			i++
		}
	}
}

func (p *jsonScan) object(i int) (int, string) {
	s := p.s
	i = skipWS(s, i)
	if p.at(i) == '}' && i < len(s) {
		return i + 1, ""
	}
	for {
		if p.at(i) != '"' || i >= len(s) {
			return 0, jsonMessage("Expecting property name enclosed in double quotes", s, i)
		}
		end, failure := p.str(i + 1)
		if failure != "" {
			return 0, failure
		}
		i = skipWS(s, end)
		if p.at(i) != ':' || i >= len(s) {
			return 0, jsonMessage("Expecting ':' delimiter", s, i)
		}
		if i, failure = p.value(skipWS(s, i+1)); failure != "" {
			return 0, failure
		}
		i = skipWS(s, i)
		switch {
		case i < len(s) && s[i] == '}':
			return i + 1, ""
		case i >= len(s) || s[i] != ',':
			return 0, jsonMessage("Expecting ',' delimiter", s, i)
		}
		i = skipWS(s, i+1)
	}
}

func (p *jsonScan) array(i int) (int, string) {
	s := p.s
	i = skipWS(s, i)
	if i < len(s) && s[i] == ']' {
		return i + 1, ""
	}
	for {
		end, failure := p.value(i)
		if failure != "" {
			return 0, failure
		}
		i = skipWS(s, end)
		switch {
		case i < len(s) && s[i] == ']':
			return i + 1, ""
		case i >= len(s) || s[i] != ',':
			return 0, jsonMessage("Expecting ',' delimiter", s, i)
		}
		i = skipWS(s, i+1)
	}
}
