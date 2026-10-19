package signalindex

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

const upperHex = "0123456789ABCDEF"

// ParseArgs parses a raw URL query string exactly like Flask's request.args (Werkzeug 3:
// urllib.parse.parse_qsl with keep_blank_values and the "werkzeug.url_quote" error
// handler): pairs split on "&" only (";" is data), empty pairs skipped, a pair without "="
// has an empty value, "+" is a space, invalid %-escapes stay literal and bytes that are not
// valid UTF-8 after unescaping stay %-encoded (upper-case hex). Go's url.ParseQuery differs
// on all of these, so the HTTP layer should use this for r.URL.RawQuery. A raw query that
// is not valid UTF-8 itself makes Werkzeug fail; such bytes are %-encoded here.
func ParseArgs(raw string) url.Values {
	values := url.Values{}
	for _, pair := range strings.Split(raw, "&") {
		if pair == "" {
			continue
		}
		name, value, _ := strings.Cut(pair, "=")
		key := unquote(name)
		values[key] = append(values[key], unquote(value))
	}
	return values
}

// unquote is urllib.parse.unquote(text.replace("+", " ")) with Werkzeug's error handler.
func unquote(text string) string {
	text = strings.ReplaceAll(text, "+", " ")
	if !strings.Contains(text, "%") {
		return text
	}
	raw := make([]byte, 0, len(text))
	for index := 0; index < len(text); index++ {
		if text[index] == '%' && index+2 < len(text) && isHex(text[index+1]) && isHex(text[index+2]) {
			raw = append(raw, hexValue(text[index+1])<<4|hexValue(text[index+2]))
			index += 2
			continue
		}
		raw = append(raw, text[index])
	}
	var out strings.Builder
	out.Grow(len(raw))
	for len(raw) > 0 {
		r, size := utf8.DecodeRune(raw)
		if r == utf8.RuneError && size <= 1 {
			out.WriteByte('%')
			out.WriteByte(upperHex[raw[0]>>4])
			out.WriteByte(upperHex[raw[0]&15])
			raw = raw[1:]
			continue
		}
		out.Write(raw[:size])
		raw = raw[size:]
	}
	return out.String()
}

func isHex(char byte) bool {
	return (char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')
}

func hexValue(char byte) byte {
	switch {
	case char >= 'a':
		return char - 'a' + 10
	case char >= 'A':
		return char - 'A' + 10
	}
	return char - '0'
}
