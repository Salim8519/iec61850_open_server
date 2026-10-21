package proto

import (
	"fmt"
	"strconv"
	"strings"
)

// OID is an object identifier as its arcs.
type OID []uint32

// ParseOID reads "1.3.6.1.2.1.1.1.0" (a leading dot is allowed).
func ParseOID(text string) (OID, error) {
	text = strings.TrimPrefix(strings.TrimSpace(text), ".")
	if text == "" {
		return nil, fmt.Errorf("empty OID")
	}
	parts := strings.Split(text, ".")
	oid := make(OID, len(parts))
	for index, part := range parts {
		arc, err := strconv.ParseUint(part, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%q is not a numeric OID", text)
		}
		oid[index] = uint32(arc)
	}
	if len(oid) < 2 || oid[0] > 2 || (oid[0] < 2 && oid[1] >= 40) {
		return nil, fmt.Errorf("%q is not a valid OID", text)
	}
	return oid, nil
}

// MustOID parses a constant OID.
func MustOID(text string) OID {
	oid, err := ParseOID(text)
	if err != nil {
		panic(err)
	}
	return oid
}

func (o OID) String() string {
	var b strings.Builder
	for index, arc := range o {
		if index > 0 {
			b.WriteByte('.')
		}
		b.WriteString(strconv.FormatUint(uint64(arc), 10))
	}
	return b.String()
}

// Compare orders OIDs lexicographically by arc (-1, 0, 1).
func (o OID) Compare(other OID) int {
	for index := 0; index < len(o) && index < len(other); index++ {
		if o[index] != other[index] {
			if o[index] < other[index] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(o) < len(other):
		return -1
	case len(o) > len(other):
		return 1
	}
	return 0
}

// HasPrefix reports whether o lies at or under prefix.
func (o OID) HasPrefix(prefix OID) bool {
	return len(o) >= len(prefix) && o[:len(prefix)].Compare(prefix) == 0
}

// Value is one SNMP value. Int holds INTEGER, Uint the unsigned types, Bytes OCTET STRING,
// Opaque and IpAddress (4 bytes), OID an OBJECT IDENTIFIER. Null and the three exceptions
// carry nothing.
type Value struct {
	Type  byte
	Int   int64
	Uint  uint64
	Bytes []byte
	OID   OID
}

// VarBind is one variable binding.
type VarBind struct {
	OID   OID
	Value Value
}

// Null is the value of a request's variable bindings.
var Null = Value{Type: TagNull}

// Exception values of a GET/GETNEXT answer (v2c and v3).
var (
	NoSuchObject   = Value{Type: TagNoSuchObj}
	NoSuchInstance = Value{Type: TagNoSuchInst}
	EndOfMibView   = Value{Type: TagEndOfView}
)

// IsException reports noSuchObject, noSuchInstance or endOfMibView.
func (v Value) IsException() bool {
	return v.Type == TagNoSuchObj || v.Type == TagNoSuchInst || v.Type == TagEndOfView
}

func encodeValue(v Value) ([]byte, error) {
	switch v.Type {
	case TagInteger:
		return intElement(v.Int), nil
	case TagOctetString, TagOpaque:
		return tlv(v.Type, v.Bytes), nil
	case TagIPAddress:
		if len(v.Bytes) != 4 {
			return nil, fmt.Errorf("IpAddress needs 4 bytes")
		}
		return tlv(TagIPAddress, v.Bytes), nil
	case TagCounter32, TagGauge32, TagTimeTicks, TagCounter64:
		return tlv(v.Type, encodeUint(v.Uint)), nil
	case TagOID:
		content, err := encodeOID(v.OID)
		if err != nil {
			return nil, err
		}
		return tlv(TagOID, content), nil
	case TagNull, TagNoSuchObj, TagNoSuchInst, TagEndOfView:
		return []byte{v.Type, 0}, nil
	}
	return nil, fmt.Errorf("unknown value type 0x%02x", v.Type)
}

func decodeValue(tag byte, content []byte) (Value, error) {
	v := Value{Type: tag}
	var err error
	switch tag {
	case TagInteger:
		v.Int, err = decodeInt(content)
	case TagOctetString, TagOpaque, TagIPAddress:
		v.Bytes = content
	case TagCounter32, TagGauge32, TagTimeTicks, TagCounter64:
		v.Uint, err = decodeUint(content)
		if err == nil && tag != TagCounter64 && v.Uint > 0xffffffff {
			err = malformed("32-bit value too large")
		}
	case TagOID:
		v.OID, err = decodeOID(content)
	case TagNull, TagNoSuchObj, TagNoSuchInst, TagEndOfView:
	default:
		err = malformed(fmt.Sprintf("unknown value tag 0x%02x", tag))
	}
	return v, err
}
