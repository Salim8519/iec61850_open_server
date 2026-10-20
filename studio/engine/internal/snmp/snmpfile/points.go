// Package snmpfile imports and exports SNMP points (ADR-0092): Net-SNMP snmpwalk output and
// the ScadaBench CSV format. It only converts text; it never opens files or sockets.
package snmpfile

import (
	"sort"
	"strconv"
	"strings"

	"studio-engine/internal/snmp/mib"
	"studio-engine/internal/snmp/model"
)

// Problem is one line that could not be used, with the reason in plain English.
type Problem struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

// sysUpTimeOID is sysUpTime.0; a simulated device keeps it running.
const sysUpTimeOID = "1.3.6.1.2.1.1.3.0"

// collector gathers points, keeps the last of duplicates and records problems.
type collector struct {
	points   []model.Point
	at       map[string]int
	problems []Problem
}

func newCollector() *collector { return &collector{at: map[string]int{}} }

func (c *collector) problem(line int, text string) {
	c.problems = append(c.problems, Problem{Line: line, Text: text})
}

func (c *collector) add(line int, p model.Point) {
	if i, dup := c.at[p.OID]; dup {
		c.problem(line, p.OID+" appears more than once; the later value is kept")
		c.points[i] = p
		return
	}
	c.at[p.OID] = len(c.points)
	c.points = append(c.points, p)
}

// sorted returns the points ordered by OID, arc by arc.
func (c *collector) sorted() []model.Point {
	sort.SliceStable(c.points, func(i, j int) bool { return mib.CompareOID(c.points[i].OID, c.points[j].OID) < 0 })
	return c.points
}

// defaultBehaviour is "uptime" for sysUpTime.0 and "fixed" for everything else.
func defaultBehaviour(oid string) string {
	if oid == sysUpTimeOID {
		return model.BehaviourUptime
	}
	return model.BehaviourFixed
}

// normalizeHex accepts "00 1A ff", "00:1a:ff", "00-1a-ff" or "001aff" and returns
// "00 1a ff". ok is false for anything else (odd digit count, non-hex characters).
func normalizeHex(s string) (string, bool) {
	var digits []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ' || c == ':' || c == '-' || c == '\t' || c == '\n' || c == '\r':
			if len(digits)%2 != 0 {
				return "", false
			}
		case isHexDigit(c):
			digits = append(digits, lower(c))
		default:
			return "", false
		}
	}
	if len(digits)%2 != 0 {
		return "", false
	}
	pairs := make([]string, 0, len(digits)/2)
	for i := 0; i < len(digits); i += 2 {
		pairs = append(pairs, string(digits[i:i+2]))
	}
	return strings.Join(pairs, " "), true
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 'a' - 'A'
	}
	return c
}

// parseUnsigned checks a decimal number against a type's upper limit and returns it in
// canonical form ("007" -> "7").
func parseUnsigned(s string, bits int) (string, bool) {
	n, err := strconv.ParseUint(strings.TrimPrefix(s, "+"), 10, bits)
	if err != nil {
		return "", false
	}
	return strconv.FormatUint(n, 10), true
}

// parseInteger checks an INTEGER (Integer32 range).
func parseInteger(s string) (string, bool) {
	n, err := strconv.ParseInt(strings.TrimPrefix(s, "+"), 10, 32)
	if err != nil {
		return "", false
	}
	return strconv.FormatInt(n, 10), true
}

// parseIPv4 checks a dotted quad and returns it canonical ("010.0.0.1" is rejected).
func parseIPv4(s string) (string, bool) {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return "", false
	}
	for _, p := range parts {
		n, err := strconv.ParseUint(p, 10, 8)
		if err != nil || strconv.FormatUint(n, 10) != p {
			return "", false
		}
	}
	return s, true
}

// enumValue finds the number of a named value ("up") in the MIB object an OID belongs to.
func enumValue(lib *mib.Library, oid, label string) (string, bool) {
	n, _ := lib.Lookup(oid)
	if n == nil {
		return "", false
	}
	for v, name := range n.Enums {
		if name == label {
			return strconv.FormatInt(v, 10), true
		}
	}
	return "", false
}

// library returns lib, or an empty library (well-known roots only) when lib is nil.
func library(lib *mib.Library) *mib.Library {
	if lib == nil {
		return mib.NewLibrary()
	}
	return lib
}
