package projectapi

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unicode/utf16"
	"unicode/utf8"
)

var errBodyEncoding = errors.New("undecodable JSON body")

// jsonText is what json.loads(bytes) decodes first: json.detect_encoding picks UTF-8
// (BOM stripped), UTF-16 or UTF-32 from a BOM or the zero-byte pattern of the first bytes.
func jsonText(b []byte) ([]byte, error) {
	switch {
	case bytes.HasPrefix(b, []byte{0, 0, 0xfe, 0xff}):
		return utf32(b[4:], binary.BigEndian)
	case bytes.HasPrefix(b, []byte{0xff, 0xfe, 0, 0}):
		return utf32(b[4:], binary.LittleEndian)
	case bytes.HasPrefix(b, []byte{0xfe, 0xff}):
		return utf16Text(b[2:], binary.BigEndian)
	case bytes.HasPrefix(b, []byte{0xff, 0xfe}):
		return utf16Text(b[2:], binary.LittleEndian)
	case bytes.HasPrefix(b, []byte{0xef, 0xbb, 0xbf}):
		return utf8Text(b[3:])
	case len(b) >= 4 && b[0] == 0:
		if b[1] != 0 {
			return utf16Text(b, binary.BigEndian)
		}
		return utf32(b, binary.BigEndian)
	case len(b) >= 4 && b[1] == 0:
		if b[2] != 0 || b[3] != 0 {
			return utf16Text(b, binary.LittleEndian)
		}
		return utf32(b, binary.LittleEndian)
	case len(b) == 2 && b[0] == 0:
		return utf16Text(b, binary.BigEndian)
	case len(b) == 2 && b[1] == 0:
		return utf16Text(b, binary.LittleEndian)
	}
	return utf8Text(b)
}

func utf8Text(b []byte) ([]byte, error) {
	if !utf8.Valid(b) {
		return nil, errBodyEncoding
	}
	return b, nil
}

func utf16Text(b []byte, order binary.ByteOrder) ([]byte, error) {
	if len(b)%2 != 0 {
		return nil, errBodyEncoding
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = order.Uint16(b[2*i:])
	}
	var out []byte
	for _, r := range utf16.Decode(units) {
		if r == utf8.RuneError {
			return nil, errBodyEncoding
		}
		out = utf8.AppendRune(out, r)
	}
	return out, nil
}

func utf32(b []byte, order binary.ByteOrder) ([]byte, error) {
	if len(b)%4 != 0 {
		return nil, errBodyEncoding
	}
	var out []byte
	for i := 0; i < len(b); i += 4 {
		r := rune(order.Uint32(b[i:]))
		if r > utf8.MaxRune || r >= 0xd800 && r <= 0xdfff {
			return nil, errBodyEncoding
		}
		out = utf8.AppendRune(out, r)
	}
	return out, nil
}
