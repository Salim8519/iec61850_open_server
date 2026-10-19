package updates

import (
	"strconv"
	"strings"
)

// AllowedHosts is versions.ALLOWED_HOSTS: a release may only point at the product's sites.
var AllowedHosts = map[string]bool{"scadabench.com": true, "www.scadabench.com": true, "download.scadabench.com": true}

const schemeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789+-."

// nfkcDelimiters are characters whose NFKC form contains "/", "?", "#", "@" or ":";
// urllib's _checknetloc refuses a non-ASCII netloc holding one of them.
const nfkcDelimiters = "／？＃＠：﹟﹫﹖﹕︓︖⁇⁈⁉" +
	"℀℁℅℆"

func isASCII(text string) bool {
	for index := 0; index < len(text); index++ {
		if text[index] >= 0x80 {
			return false
		}
	}
	return true
}

// splitNetloc is urlsplit up to the network location: "" when the URL has none or is not
// https (both make the URL unsafe).
func splitNetloc(url string) (string, bool) {
	url = strings.TrimLeft(url, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\x0b\x0c\r\x0e\x0f"+
		"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f ")
	url = strings.NewReplacer("\t", "", "\r", "", "\n", "").Replace(url)
	scheme := ""
	if colon := strings.IndexByte(url, ':'); colon > 0 && isLetter(url[0]) {
		if strings.Trim(url[:colon], schemeChars) == "" {
			scheme, url = strings.ToLower(url[:colon]), url[colon+1:]
		}
	}
	if scheme != "https" || !strings.HasPrefix(url, "//") {
		return "", false
	}
	netloc := url[2:]
	if end := strings.IndexAny(netloc, "/?#"); end >= 0 {
		netloc = netloc[:end]
	}
	return netloc, true
}

func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

// SafeURL is versions._safe_url: an https link to an allowed host with no user name and
// no port (an empty or zero port passes, as in Python); any ValueError urllib raises
// makes the link unsafe, exactly as clean_record then refuses it.
func SafeURL(value any) bool {
	text, isText := value.(string)
	if !isText {
		return false
	}
	netloc, ok := splitNetloc(text)
	if !ok || strings.ContainsAny(netloc, "[]") {
		return false
	}
	if !isASCII(netloc) && strings.ContainsAny(netloc, nfkcDelimiters) {
		return false
	}
	hostinfo := netloc
	if at := strings.LastIndexByte(netloc, '@'); at >= 0 {
		username, _, _ := strings.Cut(netloc[:at], ":")
		if username != "" {
			return false
		}
		hostinfo = netloc[at+1:]
	}
	hostname, port, _ := strings.Cut(hostinfo, ":")
	if port != "" {
		for index := 0; index < len(port); index++ {
			if port[index] < '0' || port[index] > '9' {
				return false
			}
		}
		number, err := strconv.ParseUint(port, 10, 64)
		if err != nil || number != 0 {
			return false
		}
	}
	return isASCII(hostname) && AllowedHosts[strings.ToLower(hostname)]
}
