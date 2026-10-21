package snmpfile

import (
	"errors"
	"strings"

	"studio-engine/internal/snmp/mib"
	"studio-engine/internal/snmp/model"
)

// record is one "name = value" line of a walk with the continuation lines that follow it.
type record struct {
	line int
	name string
	rest string
	more []textLine
}

type textLine struct {
	n    int
	text string
}

// skipped are Net-SNMP answers that carry no value; they are left out silently.
var skipped = []string{"No Such Object", "No Such Instance", "No more variables left", "End of MIB"}

// aliases are names Net-SNMP prints from MIBs that are not built in.
var aliases = map[string]string{
	"DISMAN-EVENT-MIB::sysUpTimeInstance": sysUpTimeOID, "sysUpTimeInstance": sysUpTimeOID,
}

// ParseWalk reads Net-SNMP snmpwalk/snmpbulkwalk output (default, -On or iso.* names) and
// returns fixed points sorted by OID; sysUpTime.0 keeps running ("uptime"). Lines that cannot
// be used are reported as problems. It fails only when no point at all was found.
func ParseWalk(text string, lib *mib.Library) ([]model.Point, []Problem, error) {
	lib = library(lib)
	c := newCollector()
	var recs []*record
	for i, raw := range strings.Split(strings.TrimPrefix(text, "\ufeff"), "\n") {
		line := strings.TrimRight(raw, "\r")
		if name, rest, ok := splitRecord(line); ok {
			recs = append(recs, &record{line: i + 1, name: name, rest: rest})
			continue
		}
		if len(recs) == 0 {
			if strings.TrimSpace(line) != "" {
				c.problem(i+1, "not a walk line (expected \"OID = TYPE: value\")")
			}
			continue
		}
		last := recs[len(recs)-1]
		last.more = append(last.more, textLine{i + 1, line})
	}
	for _, r := range recs {
		for len(r.more) > 0 && strings.TrimSpace(r.more[len(r.more)-1].text) == "" {
			r.more = r.more[:len(r.more)-1]
		}
		c.convert(lib, r)
	}
	points := c.sorted()
	if len(points) == 0 {
		return nil, c.problems, errors.New("no SNMP values found: expected snmpwalk lines such as " +
			"\"SNMPv2-MIB::sysDescr.0 = STRING: text\" or \".1.3.6.1.2.1.1.1.0 = STRING: text\"")
	}
	return points, c.problems, nil
}

// splitRecord recognises "name = rest". The name has no spaces outside quotes and holds a
// dot or "::", so continuation text such as "Version = 2" is not taken for a record.
func splitRecord(line string) (name, rest string, ok bool) {
	if line == "" || !(isAlnum(line[0]) || line[0] == '.') {
		return "", "", false
	}
	var quote byte
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quote = ch
		case ' ', '\t':
			if !strings.HasPrefix(line[i:], " =") || (i+2 < len(line) && line[i+2] != ' ') {
				return "", "", false
			}
			name = line[:i]
			if !strings.ContainsAny(name, ".:") {
				return "", "", false
			}
			if i+3 <= len(line) {
				rest = line[i+3:]
			}
			return name, rest, true
		}
	}
	return "", "", false
}

func isAlnum(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// convert turns one record into a point or a problem.
func (c *collector) convert(lib *mib.Library, r *record) {
	rest := r.rest
	for _, s := range skipped {
		if strings.HasPrefix(rest, s) {
			return
		}
	}
	oid, ok := aliases[r.name]
	if !ok {
		oid, ok = lib.Resolve(r.name)
	}
	if !ok {
		c.problem(r.line, r.name+": unknown name — import the MIB that defines it, or export the walk with -On")
		return
	}
	if strings.HasPrefix(rest, "Wrong Type") {
		if i := strings.Index(rest, "): "); i >= 0 {
			rest = rest[i+3:]
		}
	}
	p, why := c.walkValue(lib, oid, rest, r.more)
	if why != "" {
		c.problem(r.line, r.name+": "+why)
		return
	}
	p.OID, p.Behaviour = oid, defaultBehaviour(oid)
	c.add(r.line, p)
}

// unused reports continuation lines that belong to no value.
func (c *collector) unused(more []textLine) {
	for _, l := range more {
		if strings.TrimSpace(l.text) != "" {
			c.problem(l.n, "not a walk line (expected \"OID = TYPE: value\")")
		}
	}
}

// quotedString reads a "quoted" value that may continue over the following lines. Net-SNMP
// escapes quotes and backslashes inside with a backslash.
func (c *collector) quotedString(val string, more []textLine) (string, bool) {
	full, used := val, 0
	end := closingQuote(full)
	for end < 0 && used < len(more) {
		full += "\n" + more[used].text
		used++
		end = closingQuote(full)
	}
	c.unused(more[used:])
	if end < 0 {
		return "", false
	}
	var b strings.Builder
	for i := 1; i < end; i++ {
		if full[i] == '\\' && i+1 < end && (full[i+1] == '"' || full[i+1] == '\\') {
			i++
		}
		b.WriteByte(full[i])
	}
	return b.String(), true
}

// closingQuote returns the index of the quote that ends a string starting with a quote.
func closingQuote(s string) int {
	for i := 1; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '"':
			return i
		}
	}
	return -1
}
