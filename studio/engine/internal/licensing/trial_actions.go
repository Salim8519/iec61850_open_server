package licensing

import (
	"strings"
	"unicode/utf8"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// ServerHost is LicenseService.server_host: the license server's host name for messages,
// or "the license server" when the address has none.
func (s *Service) ServerHost() (string, error) {
	url, err := s.ServerURL()
	if err != nil {
		return "", err
	}
	host, err := urlHostname(url)
	if err != nil || host == "" {
		return "the license server", err
	}
	return host, nil
}

// TrialParts is LicenseService.trial_parts: the salted part hashes for the server's trial
// check, or {} when they cannot be read.
func (s *Service) TrialParts() *pyjson.Object {
	parts, err := s.options.Parts()
	if err != nil || parts == nil {
		return pyjson.NewObject()
	}
	return parts
}

// StartTrial starts (or gets back) this PC's trial; email and code (decoded JSON values,
// nil when absent) are sent only after the server asked for them.
func (s *Service) StartTrial(email, code any) (*pyjson.Object, error) {
	if err := s.requireEnforced(); err != nil {
		return nil, err
	}
	client, err := s.client()
	if err != nil {
		return nil, err
	}
	hwid, err := s.HWID()
	if err != nil {
		return nil, err
	}
	answer, err := client.StartTrial(hwid, s.TrialParts(), email, code)
	if err != nil {
		return nil, err
	}
	return s.install(answer)
}

// RequestTrialCode asks the server to email a trial code to email (any decoded JSON
// value); the answer's status is "sent", or "no_code_needed" when mail is unavailable.
func (s *Service) RequestTrialCode(email any) (*pyjson.Object, error) {
	if err := s.requireEnforced(); err != nil {
		return nil, err
	}
	address := ""
	if truthy(email) {
		address = pyv.PyStr(email)
	}
	address = pytext.Strip(address)
	if !strings.Contains(address, "@") || utf8.RuneCountInString(address) > 254 {
		return nil, &ActionError{Message: BadEmailText, Code: "bad_email"}
	}
	client, err := s.client()
	if err != nil {
		return nil, err
	}
	hwid, err := s.HWID()
	if err != nil {
		return nil, err
	}
	answer, err := client.RequestTrialCode(hwid, address)
	if err != nil {
		return nil, err
	}
	object, isObject := answer.(*pyjson.Object)
	if !isObject {
		return nil, uncaught("license server answer is not an object")
	}
	status, found := object.Get("status")
	if !found {
		status = "sent"
	}
	return pyjson.NewObject().Set("status", status), nil
}

// urlHostname is urllib.parse.urlparse(url).hostname ("" for None). An error stands for
// the ValueError of an unbalanced IPv6 bracket, which Python does not catch.
func urlHostname(url string) (string, error) {
	url = strings.TrimLeft(url, "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\x0b\x0c\r\x0e\x0f"+
		"\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f ")
	url = strings.NewReplacer("\t", "", "\r", "", "\n", "").Replace(url)
	if colon := strings.Index(url, ":"); colon > 0 && isSchemeName(url[:colon]) {
		url = url[colon+1:]
	}
	if !strings.HasPrefix(url, "//") {
		return "", nil
	}
	netloc := url[2:]
	if end := strings.IndexAny(netloc, "/?#"); end >= 0 {
		netloc = netloc[:end]
	}
	if strings.Contains(netloc, "[") != strings.Contains(netloc, "]") {
		return "", uncaught("Invalid IPv6 URL")
	}
	host := netloc[strings.LastIndex(netloc, "@")+1:]
	if _, bracketed, found := strings.Cut(host, "["); found {
		host, _, _ = strings.Cut(bracketed, "]")
	} else {
		host, _, _ = strings.Cut(host, ":")
	}
	name, zone, hasZone := strings.Cut(host, "%")
	if hasZone {
		return strings.ToLower(name) + "%" + zone, nil
	}
	return strings.ToLower(host), nil
}

// isSchemeName is urlsplit's scheme test: an ASCII letter, then letters, digits, "+-.".
func isSchemeName(text string) bool {
	for index := 0; index < len(text); index++ {
		c := text[index]
		letter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		if index == 0 && !letter {
			return false
		}
		if !letter && !(c >= '0' && c <= '9') && c != '+' && c != '-' && c != '.' {
			return false
		}
	}
	return true
}
