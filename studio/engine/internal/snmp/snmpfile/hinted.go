package snmpfile

import (
	"regexp"
	"strconv"
	"strings"

	"studio-engine/internal/snmp/mib"
	"studio-engine/internal/snmp/model"
)

// dateAndTimeText is how Net-SNMP prints SNMPv2-TC DateAndTime ("2026-10-20,14:21:5.3,+4:0").
var dateAndTimeText = regexp.MustCompile(`^(\d+)-(\d+)-(\d+),(\d+):(\d+):(\d+)\.(\d+)(?:,([+-])(\d+):(\d+))?$`)

// hintedOctets undoes the DISPLAY-HINT formatting snmpwalk applies to binary octet strings
// (PhysAddress/MacAddress "0:15:5d:a:b:c", DateAndTime, plain "00 1a") when the MIB marks the
// object as hex. It returns hex octets, or ok false when the text is not such a value.
func hintedOctets(lib *mib.Library, oid, val string) (string, bool) {
	n, _ := lib.Lookup(oid)
	if n == nil || n.Syntax != model.TypeOctets || !n.Hex {
		return "", false
	}
	v := strings.TrimSpace(val)
	if v == "" {
		return "", true
	}
	if h, ok := colonHex(v); ok {
		return h, true
	}
	if h, ok := normalizeHex(v); ok {
		return h, true
	}
	return dateAndTime(v)
}

// colonHex reads "0:15:5d:a:b:c" (one or two hex digits per byte, at least two bytes).
func colonHex(v string) (string, bool) {
	parts := strings.Split(v, ":")
	if len(parts) < 2 {
		return "", false
	}
	out := make([]byte, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.ParseUint(p, 16, 8)
		if err != nil || len(p) > 2 {
			return "", false
		}
		out = append(out, byte(n))
	}
	return hexBytes(out), true
}

// dateAndTime encodes the 8 or 11 octets of a DateAndTime (RFC 2579).
func dateAndTime(v string) (string, bool) {
	m := dateAndTimeText.FindStringSubmatch(v)
	if m == nil {
		return "", false
	}
	num := func(s string, max uint64) (byte, bool) {
		n, err := strconv.ParseUint(s, 10, 16)
		return byte(n), err == nil && n <= max
	}
	year, err := strconv.ParseUint(m[1], 10, 16)
	if err != nil {
		return "", false
	}
	out := []byte{byte(year >> 8), byte(year)}
	limits := []uint64{12, 31, 23, 59, 60, 9}
	for i, s := range m[2:8] {
		b, ok := num(s, limits[i])
		if !ok {
			return "", false
		}
		out = append(out, b)
	}
	if m[8] != "" {
		h, okH := num(m[9], 13)
		mi, okM := num(m[10], 59)
		if !okH || !okM {
			return "", false
		}
		out = append(out, m[8][0], h, mi)
	}
	return hexBytes(out), true
}
