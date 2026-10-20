package snmpfile

import (
	"strings"

	"studio-engine/internal/snmp/mib"
	"studio-engine/internal/snmp/model"
)

// walkTags are the value prefixes snmpwalk prints, longest first where one is a prefix of another.
var walkTags = []string{"Hex-STRING", "STRING", "INTEGER", "Integer32", "Gauge32", "Counter32",
	"Counter64", "Unsigned32", "UInteger32", "Timeticks", "IpAddress", "OID", "Opaque", "OPAQUE",
	"BITS", "Network Address"}

// syntaxTags guess the tag from the MIB when a line has no "TYPE:" prefix (snmpwalk -OQ).
var syntaxTags = map[string]string{
	model.TypeInteger: "INTEGER", model.TypeOID: "OID", model.TypeIPAddress: "IpAddress",
	model.TypeCounter32: "Counter32", model.TypeGauge32: "Gauge32", model.TypeTimeTicks: "Timeticks",
	model.TypeCounter64: "Counter64", model.TypeOpaque: "Opaque",
}

// cutTag splits "TYPE: value" into the tag and the value.
func cutTag(rest string) (tag, val string, ok bool) {
	for _, t := range walkTags {
		if rest == t+":" {
			return t, "", true
		}
		if strings.HasPrefix(rest, t+": ") {
			return t, rest[len(t)+2:], true
		}
	}
	return "", "", false
}

// walkValue converts the text after "OID = " to a point (without OID and behaviour). why is
// the reason when the value cannot be used.
func (c *collector) walkValue(lib *mib.Library, oid, rest string, more []textLine) (model.Point, string) {
	switch strings.TrimSpace(rest) {
	case "", `""`:
		c.unused(more)
		return model.Point{Type: model.TypeOctets}, ""
	case "NULL":
		c.unused(more)
		return model.Point{Type: model.TypeNull}, ""
	}
	tag, val, ok := cutTag(rest)
	if !ok {
		tag, val = guessTag(lib, oid), rest
		if tag == "" {
			c.unused(more)
			return model.Point{}, "value has no type (expected \"TYPE: value\" as snmpwalk prints by default)"
		}
	}
	switch tag {
	case "STRING":
		return c.stringValue(lib, oid, val, more)
	case "Hex-STRING":
		return hexValue(val, more, c)
	}
	c.unused(more)
	field := strings.TrimSpace(val)
	if i := strings.IndexAny(field, " \t"); i >= 0 && tag != "OID" && tag != "Opaque" && tag != "OPAQUE" {
		field = field[:i] // drop units such as "35 degrees Celsius"
	}
	switch tag {
	case "INTEGER", "Integer32":
		return integerValue(lib, oid, field)
	case "Gauge32", "Unsigned32", "UInteger32":
		return unsignedValue(model.TypeGauge32, field, 32)
	case "Counter32":
		return unsignedValue(model.TypeCounter32, field, 32)
	case "Counter64":
		return unsignedValue(model.TypeCounter64, field, 64)
	case "Timeticks":
		return unsignedValue(model.TypeTimeTicks, strings.Trim(field, "()"), 32)
	case "IpAddress":
		if v, ok := parseIPv4(field); ok {
			return model.Point{Type: model.TypeIPAddress, Value: v}, ""
		}
		return model.Point{}, "IpAddress \"" + field + "\" is not a dotted quad such as 10.0.0.1"
	case "OID":
		if v, ok := lib.Resolve(field); ok {
			return model.Point{Type: model.TypeOID, Value: v}, ""
		}
		return model.Point{}, "OID value " + field + ": unknown name — import the MIB that defines it, or export the walk with -On"
	case "Opaque", "OPAQUE":
		v, why := opaqueValue(field)
		return model.Point{Type: model.TypeOpaque, Value: v}, why
	case "BITS":
		return bitsValue(val)
	}
	return networkAddress(field)
}

// guessTag picks a tag from the MIB syntax of the object an OID belongs to.
func guessTag(lib *mib.Library, oid string) string {
	n, _ := lib.Lookup(oid)
	if n == nil {
		return ""
	}
	if n.Syntax == model.TypeOctets {
		return "STRING" // hex objects are recognised by hintedOctets
	}
	return syntaxTags[n.Syntax]
}

func (c *collector) stringValue(lib *mib.Library, oid, val string, more []textLine) (model.Point, string) {
	if len(more) == 0 {
		if h, ok := hintedOctets(lib, oid, val); ok {
			return model.Point{Type: model.TypeOctets, Value: h, Hex: true}, ""
		}
	}
	if strings.HasPrefix(val, "\"") {
		s, ok := c.quotedString(val, more)
		if !ok {
			return model.Point{}, "STRING has no closing quote"
		}
		return model.Point{Type: model.TypeOctets, Value: s}, ""
	}
	lines := []string{val}
	for _, l := range more {
		lines = append(lines, l.text)
	}
	return model.Point{Type: model.TypeOctets, Value: strings.Join(lines, "\n")}, ""
}

// hexValue joins "Hex-STRING: 00 1A" with continuation lines of hex pairs.
func hexValue(val string, more []textLine, c *collector) (model.Point, string) {
	all := val
	for i, l := range more {
		if _, ok := normalizeHex(strings.TrimSpace(l.text)); !ok {
			c.unused(more[i:])
			break
		}
		all += " " + l.text
	}
	v, ok := normalizeHex(strings.TrimSpace(all))
	if !ok {
		return model.Point{}, "Hex-STRING holds characters that are not hex pairs"
	}
	return model.Point{Type: model.TypeOctets, Value: v, Hex: true}, ""
}

func integerValue(lib *mib.Library, oid, field string) (model.Point, string) {
	if open := strings.LastIndexByte(field, '('); open > 0 && strings.HasSuffix(field, ")") {
		field = field[open+1 : len(field)-1] // "up(1)"
	}
	if v, ok := parseInteger(field); ok {
		return model.Point{Type: model.TypeInteger, Value: v}, ""
	}
	if v, ok := enumValue(lib, oid, field); ok {
		return model.Point{Type: model.TypeInteger, Value: v}, ""
	}
	return model.Point{}, "INTEGER \"" + field + "\" is not a number from -2147483648 to 2147483647 " +
		"nor a named value of this object — load the MIB that defines it, or export the walk with -On"
}

func unsignedValue(typ, field string, bits int) (model.Point, string) {
	if v, ok := parseUnsigned(field, bits); ok {
		return model.Point{Type: typ, Value: v}, ""
	}
	limit := "4294967295"
	if bits == 64 {
		limit = "18446744073709551615"
	}
	return model.Point{}, typ + " \"" + field + "\" is not a whole number from 0 to " + limit
}

// bitsValue keeps the hex octets of "BITS: 80 00 foo(0)".
func bitsValue(val string) (model.Point, string) {
	var pairs []string
	for _, f := range strings.Fields(val) {
		if len(f) != 2 || !isHexDigit(f[0]) || !isHexDigit(f[1]) {
			break
		}
		pairs = append(pairs, f)
	}
	v, _ := normalizeHex(strings.Join(pairs, " "))
	return model.Point{Type: model.TypeOctets, Value: v, Hex: true}, ""
}

// networkAddress converts SMIv1 "Network Address: 0A:00:00:01" to an IpAddress.
func networkAddress(field string) (model.Point, string) {
	h, ok := normalizeHex(field)
	parts := strings.Fields(h)
	if !ok || len(parts) != 4 {
		return model.Point{}, "Network Address \"" + field + "\" is not four hex bytes such as 0A:00:00:01"
	}
	ip := make([]string, 4)
	for i, p := range parts {
		ip[i] = hexByteDecimal(p)
	}
	return model.Point{Type: model.TypeIPAddress, Value: strings.Join(ip, ".")}, ""
}
