package httpapi

import (
	"net/netip"
	"strings"

	"studio-engine/internal/apperr"
)

// urlNetloc is urllib.parse.urlsplit(url).netloc of CPython 3.12: leading C0 controls and
// spaces and every tab/CR/LF are removed, an optional scheme is skipped, and the netloc is
// what follows "//" up to "/", "?" or "#". Unbalanced or invalid brackets raise the
// ValueError Flask answered with 400.
func urlNetloc(url string) (string, error) {
	url = strings.TrimLeft(url, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f"+
		"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f ")
	url = strings.NewReplacer("\t", "", "\r", "", "\n", "").Replace(url)
	if colon := strings.IndexByte(url, ':'); colon > 0 && isASCIILetter(url[0]) && isScheme(url[:colon]) {
		url = url[colon+1:]
	}
	if !strings.HasPrefix(url, "//") {
		return "", nil
	}
	rest := url[2:]
	end := len(rest)
	if index := strings.IndexAny(rest, "/?#"); index >= 0 {
		end = index
	}
	netloc := rest[:end]
	open, close := strings.Contains(netloc, "["), strings.Contains(netloc, "]")
	if open != close {
		return "", apperr.ValueError("Invalid IPv6 URL")
	}
	if open {
		_, after, _ := strings.Cut(netloc, "[")
		host, _, _ := strings.Cut(after, "]")
		if err := checkBracketedHost(host); err != nil {
			return "", err
		}
	}
	return netloc, nil
}

func checkBracketedHost(host string) error {
	if strings.HasPrefix(host, "v") {
		return nil // IPvFuture: CPython checks a pattern; not reachable from browsers
	}
	address, err := netip.ParseAddr(host)
	if err != nil {
		return apperr.ValueError("'" + host + "' does not appear to be an IPv4 or IPv6 address")
	}
	if address.Is4() {
		return apperr.ValueError("An IPv4 address cannot be in brackets")
	}
	return nil
}

func isASCIILetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

func isScheme(text string) bool {
	for i := 0; i < len(text); i++ {
		c := text[i]
		if !isASCIILetter(c) && !(c >= '0' && c <= '9') && c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}
