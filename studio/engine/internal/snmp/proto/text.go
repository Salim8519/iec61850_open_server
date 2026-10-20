package proto

import (
	"encoding/hex"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"unicode/utf8"

	"studio-engine/internal/snmp/model"
)

var typeTags = map[string]byte{model.TypeInteger: TagInteger, model.TypeOctets: TagOctetString,
	model.TypeOID: TagOID, model.TypeIPAddress: TagIPAddress, model.TypeCounter32: TagCounter32,
	model.TypeGauge32: TagGauge32, model.TypeTimeTicks: TagTimeTicks, model.TypeCounter64: TagCounter64,
	model.TypeOpaque: TagOpaque, model.TypeNull: TagNull}

// TypeName is the model type of a BER tag ("" for the exceptions).
func TypeName(tag byte) string {
	for name, value := range typeTags {
		if value == tag {
			return name
		}
	}
	return ""
}

// ParseValue turns a point's text into a value of its type; the error text is for the user.
func ParseValue(kind, text string, isHex bool) (Value, error) {
	tag, known := typeTags[kind]
	if !known {
		return Value{}, fmt.Errorf("unknown type %q", kind)
	}
	text = strings.TrimSpace(text)
	v := Value{Type: tag}
	switch tag {
	case TagInteger:
		number, err := strconv.ParseInt(text, 10, 32)
		if err != nil {
			return v, fmt.Errorf("%q is not a whole number from -2147483648 to 2147483647", text)
		}
		v.Int = number
	case TagCounter32, TagGauge32, TagTimeTicks, TagCounter64:
		size := 32
		if tag == TagCounter64 {
			size = 64
		}
		number, err := strconv.ParseUint(text, 10, size)
		if err != nil {
			return v, fmt.Errorf("%q is not a whole number from 0 to %d", text, uint64(1)<<size-1)
		}
		v.Uint = number
	case TagOctetString, TagOpaque:
		if isHex || tag == TagOpaque {
			data, err := ParseHex(text)
			if err != nil {
				return v, err
			}
			v.Bytes = data
		} else {
			v.Bytes = []byte(text)
		}
	case TagIPAddress:
		address, err := netip.ParseAddr(text)
		if err != nil || !address.Is4() {
			return v, fmt.Errorf("%q is not an IPv4 address such as 192.168.1.10", text)
		}
		four := address.As4()
		v.Bytes = four[:]
	case TagOID:
		oid, err := ParseOID(text)
		if err != nil {
			return v, err
		}
		v.OID = oid
	}
	return v, nil
}

// ParseHex reads "00 1a ff", "00:1A:FF", "001aff" or "0x001aff".
func ParseHex(text string) ([]byte, error) {
	clean := strings.NewReplacer(" ", "", ":", "", "-", "", "\n", "", "\r", "", "\t", "").
		Replace(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(text)), "0x"))
	data, err := hex.DecodeString(clean)
	if err != nil {
		return nil, fmt.Errorf("%q is not hex bytes such as 00 1a ff", text)
	}
	return data, nil
}

// FormatHex writes bytes as "00 1a ff".
func FormatHex(data []byte) string {
	parts := make([]string, len(data))
	for index, b := range data {
		parts[index] = hex.EncodeToString([]byte{b})
	}
	return strings.Join(parts, " ")
}

// ValueText is a value as a point's type, text and hex flag. Octets that are not printable
// UTF-8 come back as hex.
func ValueText(v Value) (kind, text string, isHex bool) {
	switch v.Type {
	case TagInteger:
		return model.TypeInteger, strconv.FormatInt(v.Int, 10), false
	case TagCounter32, TagGauge32, TagTimeTicks, TagCounter64:
		return TypeName(v.Type), strconv.FormatUint(v.Uint, 10), false
	case TagOctetString:
		if Printable(v.Bytes) {
			return model.TypeOctets, string(v.Bytes), false
		}
		return model.TypeOctets, FormatHex(v.Bytes), true
	case TagOpaque:
		return model.TypeOpaque, FormatHex(v.Bytes), true
	case TagIPAddress:
		if len(v.Bytes) == 4 {
			return model.TypeIPAddress, netip.AddrFrom4([4]byte(v.Bytes)).String(), false
		}
		return model.TypeIPAddress, FormatHex(v.Bytes), true
	case TagOID:
		return model.TypeOID, v.OID.String(), false
	case TagNoSuchObj:
		return "", "No such object", false
	case TagNoSuchInst:
		return "", "No such instance", false
	case TagEndOfView:
		return "", "End of MIB view", false
	}
	return model.TypeNull, "", false
}

// Printable reports text a person can read: valid UTF-8 without control characters other
// than tab, CR and LF (a trailing NUL, common in vendor strings, is allowed).
func Printable(data []byte) bool {
	if len(data) > 0 && data[len(data)-1] == 0 {
		data = data[:len(data)-1]
	}
	if !utf8.Valid(data) {
		return false
	}
	for _, r := range string(data) {
		if r < 0x20 && r != '\t' && r != '\r' && r != '\n' || r == 0x7f {
			return false
		}
	}
	return true
}
