package snmpfile

import (
	"math"
	"strconv"
	"strings"

	"studio-engine/internal/snmp/mib"
	"studio-engine/internal/snmp/model"
)

// checkValue validates a value for a point type and returns it in the form the project
// stores: decimal numbers, dotted OIDs and addresses, "00 1a ff" for hex. why explains a
// value that does not fit the type.
func checkValue(lib *mib.Library, oid, typ, value string, hex bool) (string, string) {
	v := strings.TrimSpace(value)
	switch typ {
	case model.TypeInteger:
		if n, ok := parseInteger(v); ok {
			return n, ""
		}
		if n, ok := enumValue(lib, oid, v); ok {
			return n, ""
		}
		return "", "integer \"" + v + "\" is not a number from -2147483648 to 2147483647 or a named value of this object"
	case model.TypeCounter32, model.TypeGauge32, model.TypeTimeTicks:
		if n, ok := parseUnsigned(v, 32); ok {
			return n, ""
		}
		return "", typ + " \"" + v + "\" is not a whole number from 0 to 4294967295"
	case model.TypeCounter64:
		if n, ok := parseUnsigned(v, 64); ok {
			return n, ""
		}
		return "", "counter64 \"" + v + "\" is not a whole number from 0 to 18446744073709551615"
	case model.TypeIPAddress:
		if ip, ok := parseIPv4(v); ok {
			return ip, ""
		}
		return "", "ipaddress \"" + v + "\" is not a dotted quad such as 10.0.0.1"
	case model.TypeOID:
		if mib.ValidOID(v) {
			return strings.TrimPrefix(v, "."), ""
		}
		if o, ok := lib.Resolve(v); ok {
			return o, ""
		}
		return "", "oid value \"" + v + "\" is not a dotted number such as 1.3.6.1.4.1.99999 or a known name"
	case model.TypeOpaque:
		if h, ok := normalizeHex(v); ok {
			return h, ""
		}
		return "", "opaque \"" + v + "\" is not hex pairs such as \"9f 78 04 3f c0 00 00\""
	case model.TypeNull:
		if v != "" {
			return "", "a null point has no value; leave the value empty"
		}
		return "", ""
	}
	if hex {
		if h, ok := normalizeHex(v); ok {
			return h, ""
		}
		return "", "hex value \"" + v + "\" is not hex pairs such as \"00 1a ff\""
	}
	return value, ""
}

// validType reports whether typ is one of the model point types.
func validType(typ string) bool {
	for _, t := range model.Types {
		if t == typ {
			return true
		}
	}
	return false
}

// checkBehaviour checks that a behaviour exists and suits the point type.
func checkBehaviour(behaviour, typ string) string {
	switch behaviour {
	case model.BehaviourFixed:
		return ""
	case model.BehaviourUptime:
		if typ != model.TypeTimeTicks {
			return "behaviour \"uptime\" needs type timeticks"
		}
		return ""
	case model.BehaviourCounter:
		switch typ {
		case model.TypeInteger, model.TypeCounter32, model.TypeGauge32, model.TypeTimeTicks, model.TypeCounter64:
			return ""
		}
		return "behaviour \"counter\" needs a number type (integer, counter32, gauge32, timeticks or counter64)"
	}
	return "behaviour \"" + behaviour + "\" is unknown; use fixed, uptime or counter"
}

// parseBool accepts true/false, yes/no, 1/0 and empty (false), in any letter case.
func parseBool(s string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "yes", "1":
		return true, true
	case "false", "no", "0", "":
		return false, true
	}
	return false, false
}

// parseRate reads the counter rate per second; empty means 0.
func parseRate(s string) (float64, bool) {
	if s == "" {
		return 0, true
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}
