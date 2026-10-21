// Package proto encodes and decodes SNMP messages (RFC 1157, RFC 3416, RFC 3412) and implements
// the user-based security model (RFC 3414, RFC 3826, RFC 7860) for studio_snmp (ADR-0092).
// It is pure: no sockets, no clocks, no files.
package proto

import (
	"errors"
	"fmt"
)

// BER tags SNMP uses.
const (
	TagInteger     byte = 0x02
	TagOctetString byte = 0x04
	TagNull        byte = 0x05
	TagOID         byte = 0x06
	TagSequence    byte = 0x30
	TagIPAddress   byte = 0x40
	TagCounter32   byte = 0x41
	TagGauge32     byte = 0x42
	TagTimeTicks   byte = 0x43
	TagOpaque      byte = 0x44
	TagCounter64   byte = 0x46
	TagNoSuchObj   byte = 0x80
	TagNoSuchInst  byte = 0x81
	TagEndOfView   byte = 0x82
)

// ErrMalformed is returned for any packet that is not valid BER for SNMP.
var ErrMalformed = errors.New("malformed SNMP packet")

func malformed(what string) error { return fmt.Errorf("%w: %s", ErrMalformed, what) }

// appendLength writes a definite BER length.
func appendLength(dst []byte, length int) []byte {
	switch {
	case length < 0x80:
		return append(dst, byte(length))
	case length <= 0xff:
		return append(dst, 0x81, byte(length))
	case length <= 0xffff:
		return append(dst, 0x82, byte(length>>8), byte(length))
	case length <= 0xffffff:
		return append(dst, 0x83, byte(length>>16), byte(length>>8), byte(length))
	default:
		return append(dst, 0x84, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
	}
}

// tlv returns tag, length and content as one element.
func tlv(tag byte, content []byte) []byte {
	out := make([]byte, 0, len(content)+6)
	out = append(out, tag)
	out = appendLength(out, len(content))
	return append(out, content...)
}

// sequence joins elements into one constructed element with tag.
func sequence(tag byte, parts ...[]byte) []byte {
	size := 0
	for _, part := range parts {
		size += len(part)
	}
	content := make([]byte, 0, size)
	for _, part := range parts {
		content = append(content, part...)
	}
	return tlv(tag, content)
}

// encodeInt is a minimal two's-complement INTEGER content.
func encodeInt(value int64) []byte {
	out := []byte{}
	for {
		out = append([]byte{byte(value)}, out...)
		next := value >> 8
		if (next == 0 && out[0]&0x80 == 0) || (next == -1 && out[0]&0x80 != 0) {
			return out
		}
		value = next
	}
}

// encodeUint is an unsigned INTEGER content (Counter32, Gauge32, TimeTicks, Counter64).
func encodeUint(value uint64) []byte {
	out := []byte{}
	for {
		out = append([]byte{byte(value)}, out...)
		value >>= 8
		if value == 0 {
			if out[0]&0x80 != 0 {
				out = append([]byte{0}, out...)
			}
			return out
		}
	}
}

func intElement(value int64) []byte { return tlv(TagInteger, encodeInt(value)) }

func octets(value []byte) []byte { return tlv(TagOctetString, value) }

// encodeOID is the content of an OBJECT IDENTIFIER (at least two arcs; shorter ones become 0.0).
func encodeOID(oid OID) ([]byte, error) {
	if len(oid) < 2 {
		oid = append(append(OID{}, oid...), 0, 0)[:2]
	}
	if oid[0] > 2 || (oid[0] < 2 && oid[1] >= 40) {
		return nil, fmt.Errorf("invalid OID %s", oid)
	}
	out := appendBase128(nil, uint64(oid[0])*40+uint64(oid[1]))
	for _, arc := range oid[2:] {
		out = appendBase128(out, uint64(arc))
	}
	return out, nil
}

func appendBase128(dst []byte, value uint64) []byte {
	var buffer [10]byte
	at := len(buffer) - 1
	buffer[at] = byte(value & 0x7f)
	for value >>= 7; value > 0; value >>= 7 {
		at--
		buffer[at] = byte(value&0x7f) | 0x80
	}
	return append(dst, buffer[at:]...)
}

// reader walks BER elements of one buffer. Sub-slices share the packet's array, so the
// position of an element in the packet is known from its capacity (see offsetIn).
type reader struct{ data []byte }

// next returns the tag and content of the next element.
func (r *reader) next() (byte, []byte, error) {
	if len(r.data) < 2 {
		return 0, nil, malformed("truncated element")
	}
	tag, first := r.data[0], r.data[1]
	at, length := 2, int(first)
	if first&0x80 != 0 {
		count := int(first & 0x7f)
		if count == 0 || count > 4 || len(r.data) < 2+count {
			return 0, nil, malformed("bad length")
		}
		length = 0
		for _, b := range r.data[2 : 2+count] {
			length = length<<8 | int(b)
		}
		at = 2 + count
	}
	if length < 0 || len(r.data)-at < length {
		return 0, nil, malformed("length beyond packet")
	}
	content := r.data[at : at+length] // full capacity kept: offsetIn needs it; never appended to
	r.data = r.data[at+length:]
	return tag, content, nil
}

// expect reads the next element and checks its tag.
func (r *reader) expect(tag byte, what string) ([]byte, error) {
	got, content, err := r.next()
	if err != nil {
		return nil, err
	}
	if got != tag {
		return nil, malformed(fmt.Sprintf("%s: tag 0x%02x, want 0x%02x", what, got, tag))
	}
	return content, nil
}

func (r *reader) integer(what string) (int64, error) {
	content, err := r.expect(TagInteger, what)
	if err != nil {
		return 0, err
	}
	return decodeInt(content)
}

func decodeInt(content []byte) (int64, error) {
	if len(content) == 0 || len(content) > 8 {
		return 0, malformed("integer size")
	}
	value := int64(int8(content[0]))
	for _, b := range content[1:] {
		value = value<<8 | int64(b)
	}
	return value, nil
}

func decodeUint(content []byte) (uint64, error) {
	if len(content) == 0 || len(content) > 9 || (len(content) == 9 && content[0] != 0) {
		return 0, malformed("unsigned size")
	}
	var value uint64
	for _, b := range content {
		value = value<<8 | uint64(b)
	}
	return value, nil
}

func decodeOID(content []byte) (OID, error) {
	if len(content) == 0 {
		return nil, malformed("empty OID")
	}
	oid := OID{}
	var arc uint64
	for index, b := range content {
		arc = arc<<7 | uint64(b&0x7f)
		if arc > 0xffffffff+80 {
			return nil, malformed("OID arc too large")
		}
		if b&0x80 != 0 {
			if index == len(content)-1 {
				return nil, malformed("OID ends inside an arc")
			}
			continue
		}
		if len(oid) == 0 {
			first := min(arc/40, 2)
			oid = append(oid, uint32(first), uint32(arc-first*40))
		} else {
			oid = append(oid, uint32(arc))
		}
		arc = 0
	}
	return oid, nil
}

// offsetIn is where part starts inside whole; part must be a sub-slice of whole.
func offsetIn(whole, part []byte) int { return cap(whole) - cap(part) }
