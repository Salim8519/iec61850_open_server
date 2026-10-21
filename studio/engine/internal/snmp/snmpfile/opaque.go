package snmpfile

import (
	"encoding/binary"
	"math"
	"strconv"
	"strings"
)

// Net-SNMP wraps special values in an Opaque as BER with these tags (after 0x9f).
const (
	opaqueCounter64 = 0x76
	opaqueFloat     = 0x78
	opaqueDouble    = 0x79
	opaqueInt64     = 0x7a
	opaqueUInt64    = 0x7b
)

// opaqueValue converts "Opaque: Float: 1.5", "Opaque: UInt64: 7" or raw "9F 78 04 ..." to the
// hex octets of the Opaque. why explains a value that cannot be converted.
func opaqueValue(field string) (string, string) {
	kind, num, special := strings.Cut(field, ": ")
	if !special {
		if v, ok := normalizeHex(field); ok {
			return v, ""
		}
		return "", "Opaque \"" + field + "\" is neither hex octets nor a Float/Double/Int64/UInt64 value"
	}
	num = strings.Fields(num + " ")[0]
	var content []byte
	tag := 0
	switch kind {
	case "Float":
		f, err := strconv.ParseFloat(num, 32)
		if err != nil {
			return "", "Opaque Float \"" + num + "\" is not a number"
		}
		content = binary.BigEndian.AppendUint32(nil, math.Float32bits(float32(f)))
		tag = opaqueFloat
	case "Double":
		f, err := strconv.ParseFloat(num, 64)
		if err != nil {
			return "", "Opaque Double \"" + num + "\" is not a number"
		}
		content = binary.BigEndian.AppendUint64(nil, math.Float64bits(f))
		tag = opaqueDouble
	case "Counter64", "UInt64", "U64":
		n, err := strconv.ParseUint(num, 10, 64)
		if err != nil {
			return "", "Opaque " + kind + " \"" + num + "\" is not a whole number from 0 to 18446744073709551615"
		}
		content, tag = berUnsigned(n), opaqueUInt64
		if kind == "Counter64" {
			tag = opaqueCounter64
		}
	case "Int64", "I64":
		n, err := strconv.ParseInt(num, 10, 64)
		if err != nil {
			return "", "Opaque " + kind + " \"" + num + "\" is not a 64-bit whole number"
		}
		content, tag = berSigned(n), opaqueInt64
	default:
		return "", "Opaque " + kind + " values are not supported — export the walk with -Ox for hex"
	}
	raw := append([]byte{0x9f, byte(tag), byte(len(content))}, content...)
	return hexBytes(raw), ""
}

// berUnsigned encodes a whole number in the fewest octets, with a leading zero when the
// top bit is set (BER INTEGER content).
func berUnsigned(n uint64) []byte {
	b := binary.BigEndian.AppendUint64(nil, n)
	for len(b) > 1 && b[0] == 0 && b[1]&0x80 == 0 {
		b = b[1:]
	}
	if b[0]&0x80 != 0 {
		b = append([]byte{0}, b...)
	}
	return b
}

// berSigned encodes a signed number as minimal two's complement octets.
func berSigned(n int64) []byte {
	b := binary.BigEndian.AppendUint64(nil, uint64(n))
	for len(b) > 1 && ((b[0] == 0 && b[1]&0x80 == 0) || (b[0] == 0xff && b[1]&0x80 != 0)) {
		b = b[1:]
	}
	return b
}

func hexBytes(b []byte) string {
	const digits = "0123456789abcdef"
	parts := make([]string, len(b))
	for i, c := range b {
		parts[i] = string([]byte{digits[c>>4], digits[c&15]})
	}
	return strings.Join(parts, " ")
}

// hexByteDecimal turns one hex pair ("0a") into its decimal value ("10").
func hexByteDecimal(pair string) string {
	n, _ := strconv.ParseUint(pair, 16, 8)
	return strconv.FormatUint(n, 10)
}
