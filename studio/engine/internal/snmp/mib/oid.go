package mib

import (
	"strconv"
	"strings"
)

// roots are the well-known OIDs every library knows, even without SNMPv2-SMI loaded.
var roots = map[string]string{
	"ccitt": "0", "iso": "1", "joint-iso-ccitt": "2", "zeroDotZero": "0.0",
	"org": "1.3", "dod": "1.3.6", "internet": "1.3.6.1",
	"directory": "1.3.6.1.1", "mgmt": "1.3.6.1.2", "mib-2": "1.3.6.1.2.1",
	"transmission": "1.3.6.1.2.1.10", "experimental": "1.3.6.1.3",
	"private": "1.3.6.1.4", "enterprises": "1.3.6.1.4.1", "security": "1.3.6.1.5",
	"snmpV2": "1.3.6.1.6", "snmpDomains": "1.3.6.1.6.1", "snmpProxys": "1.3.6.1.6.2",
	"snmpModules": "1.3.6.1.6.3",
}

// CompareOID orders dotted numeric OIDs arc by arc (so 1.3.6.1.2 sorts before 1.3.6.1.10).
// It returns -1, 0 or +1. Arcs that are not numbers compare as text.
func CompareOID(a, b string) int {
	a, b = strings.TrimPrefix(a, "."), strings.TrimPrefix(b, ".")
	for a != "" || b != "" {
		if a == "" {
			return -1
		}
		if b == "" {
			return 1
		}
		var x, y string
		x, a, _ = strings.Cut(a, ".")
		y, b, _ = strings.Cut(b, ".")
		if c := compareArc(x, y); c != 0 {
			return c
		}
	}
	return 0
}

func compareArc(x, y string) int {
	nx, ex := strconv.ParseUint(x, 10, 64)
	ny, ey := strconv.ParseUint(y, 10, 64)
	switch {
	case ex == nil && ey == nil && nx != ny:
		if nx < ny {
			return -1
		}
		return 1
	case ex == nil && ey == nil:
		return 0
	}
	return strings.Compare(x, y)
}

// ValidOID reports whether s is a dotted numeric OID with at least two arcs, each arc
// fitting 32 bits (a leading dot is allowed).
func ValidOID(s string) bool {
	parts := strings.Split(strings.TrimPrefix(s, "."), ".")
	if len(parts) < 2 {
		return false
	}
	for _, p := range parts {
		if !numericArc(p) {
			return false
		}
	}
	return true
}

func numericArc(p string) bool {
	if p == "" || len(p) > 10 {
		return false
	}
	_, err := strconv.ParseUint(p, 10, 32)
	return err == nil
}

// numericOID reports whether s (without a leading dot) is all numeric arcs.
func numericOID(s string) bool {
	for _, p := range strings.Split(s, ".") {
		if !numericArc(p) {
			return false
		}
	}
	return true
}

// splitName splits "label.rest" where label may hold hyphens ("mib-2.1") and the rest is
// the instance part ("3", "3.\"public\"").
func splitName(s string) (label, rest string) {
	if i := strings.IndexByte(s, '.'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

// instanceArcs converts a Net-SNMP instance suffix to numeric arcs: numbers stay, "text"
// becomes length + bytes, 'text' (IMPLIED) becomes bytes.
func instanceArcs(s string) (string, bool) {
	var out []string
	for s != "" {
		switch s[0] {
		case '"', '\'':
			end := strings.IndexByte(s[1:], s[0])
			if end < 0 {
				return "", false
			}
			text := s[1 : end+1]
			if s[0] == '"' {
				out = append(out, strconv.Itoa(len(text)))
			}
			for i := 0; i < len(text); i++ {
				out = append(out, strconv.Itoa(int(text[i])))
			}
			s = s[end+2:]
		default:
			var arc string
			arc, s, _ = strings.Cut(s, ".")
			if !numericArc(arc) {
				return "", false
			}
			out = append(out, arc)
			continue
		}
		if s != "" {
			if s[0] != '.' {
				return "", false
			}
			s = s[1:]
		}
	}
	return strings.Join(out, "."), true
}
