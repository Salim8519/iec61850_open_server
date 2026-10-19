package httpapi

import (
	"net/http"
	"strings"

	"studio-engine/internal/pycompat"
)

// Header returns a request header as Flask saw it: absent headers report false, values are
// latin-1 decoded (WSGI), several lines of one header are joined with ", ".
func Header(r *http.Request, name string) (string, bool) {
	values, found := r.Header[http.CanonicalHeaderKey(name)]
	if !found {
		return "", false
	}
	return Latin1(strings.Join(values, ", ")), true
}

// Latin1 decodes raw header bytes as ISO-8859-1, as WSGI servers do.
func Latin1(raw string) string {
	ascii := true
	for i := 0; i < len(raw); i++ {
		if raw[i] >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return raw
	}
	runes := make([]rune, len(raw))
	for i := 0; i < len(raw); i++ {
		runes[i] = rune(raw[i])
	}
	return string(runes)
}

// Arg is request.args.get(name) of Werkzeug 3.1: the query string must be UTF-8 (else
// the UnicodeDecodeError, a ValueError, that Flask answered with 400); it is split on '&'
// as parse_qsl(keep_blank_values=True) and the first value of name is returned.
func Arg(r *http.Request, name string) (string, bool, error) {
	if _, err := pycompat.DecodeUTF8([]byte(r.URL.RawQuery)); err != nil {
		return "", false, err
	}
	for _, part := range strings.Split(r.URL.RawQuery, "&") {
		if part == "" {
			continue
		}
		key, value, _ := strings.Cut(part, "=")
		if unquotePlus(key) == name {
			return unquotePlus(value), true, nil
		}
	}
	return "", false, nil
}

// unquotePlus is urllib.parse.unquote_plus with Werkzeug's "werkzeug.url_quote" error
// handler: '+' is a space, %XX escapes are decoded (invalid escapes stay as written), and
// bytes that are not UTF-8 are written back as uppercase %XX.
func unquotePlus(text string) string {
	text = strings.ReplaceAll(text, "+", " ")
	if !strings.Contains(text, "%") {
		return text
	}
	out := make([]byte, 0, len(text))
	for i := 0; i < len(text); i++ {
		if text[i] == '%' && i+2 < len(text) && isHex(text[i+1]) && isHex(text[i+2]) {
			out = append(out, unhex(text[i+1])<<4|unhex(text[i+2]))
			i += 2
			continue
		}
		out = append(out, text[i])
	}
	return pycompat.DecodeUTF8With(out, func(invalid []byte) string {
		quoted := make([]byte, 0, 3*len(invalid))
		for _, b := range invalid {
			quoted = append(quoted, '%', "0123456789ABCDEF"[b>>4], "0123456789ABCDEF"[b&0xf])
		}
		return string(quoted)
	})
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func unhex(c byte) byte {
	switch {
	case c >= 'a':
		return c - 'a' + 10
	case c >= 'A':
		return c - 'A' + 10
	}
	return c - '0'
}
