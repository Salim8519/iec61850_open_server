package httpapi

import (
	"fmt"
	"strings"
)

const tokenChars = "!#$%&'*+-.0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ^_`abcdefghijklmnopqrstuvwxyz|~"

// ContentDisposition is the header Werkzeug's send_file writes: kind is "inline" or
// "attachment"; an ASCII filename is written bare when it is a token, else quoted; a
// non-ASCII name adds RFC 5987 filename*=UTF-8''... with an ASCII fallback. (Werkzeug
// NFKD-normalises the fallback, so "é" becomes "e"; here non-ASCII is dropped.)
func ContentDisposition(kind, filename string) string {
	if isASCII(filename) {
		return kind + "; filename=" + quoteHeaderValue(filename)
	}
	var simple strings.Builder
	for i := 0; i < len(filename); i++ {
		if filename[i] < 0x80 {
			simple.WriteByte(filename[i])
		}
	}
	return kind + "; filename=" + quoteHeaderValue(simple.String()) + "; filename*=UTF-8''" +
		percentQuote(filename, "!#$&+^`|")
}

// quoteHeaderValue is werkzeug.http.quote_header_value.
func quoteHeaderValue(value string) string {
	if value == "" {
		return `""`
	}
	token := true
	for i := 0; i < len(value); i++ {
		if !strings.ContainsRune(tokenChars, rune(value[i])) {
			token = false
			break
		}
	}
	if token {
		return value
	}
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(value) + `"`
}

// percentQuote is urllib.parse.quote(text, safe): letters, digits, "_.-~", "/" and safe
// stay; every other UTF-8 byte becomes %XX.
func percentQuote(text, safe string) string {
	var out strings.Builder
	for i := 0; i < len(text); i++ {
		c := text[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			strings.IndexByte("_.-~/", c) >= 0 || strings.IndexByte(safe, c) >= 0 {
			out.WriteByte(c)
			continue
		}
		fmt.Fprintf(&out, "%%%02X", c)
	}
	return out.String()
}
