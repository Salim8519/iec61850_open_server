package archive

import (
	"fmt"
	"unicode/utf8"

	"studio-engine/internal/apperr"
)

// pyText converts the bytes given to json.loads into UTF-8 text exactly as
// json.loads(bytes) does: json.detect_encoding picks UTF-32/UTF-16 by BOM or by zero bytes,
// a UTF-8 BOM is removed ("utf-8-sig"), and the bytes are decoded with "surrogatepass".
// A decoding failure is Python's UnicodeDecodeError, a ValueError that archive.decode does
// not turn into "Invalid project archive". A lone surrogate, which Python keeps in the str,
// becomes U+FFFD - the same value pyjson.Decode gives a "\ud800" escape.
func pyText(data []byte) ([]byte, error) {
	switch {
	case hasPrefix(data, 0, 0, 0xfe, 0xff):
		return decodeWide(data[4:], 4, true, "utf-32", 4)
	case hasPrefix(data, 0xff, 0xfe, 0, 0):
		return decodeWide(data[4:], 4, false, "utf-32", 4)
	case hasPrefix(data, 0xfe, 0xff):
		return decodeWide(data[2:], 2, true, "utf-16", 2)
	case hasPrefix(data, 0xff, 0xfe):
		return decodeWide(data[2:], 2, false, "utf-16", 2)
	case hasPrefix(data, 0xef, 0xbb, 0xbf):
		return utf8Text(data[3:])
	}
	if len(data) >= 4 {
		switch {
		case data[0] == 0 && data[1] != 0:
			return decodeWide(data, 2, true, "utf-16-be", 0)
		case data[0] == 0:
			return decodeWide(data, 4, true, "utf-32-be", 0)
		case data[1] == 0 && (data[2] != 0 || data[3] != 0):
			return decodeWide(data, 2, false, "utf-16-le", 0)
		case data[1] == 0:
			return decodeWide(data, 4, false, "utf-32-le", 0)
		}
	} else if len(data) == 2 {
		switch {
		case data[0] == 0:
			return decodeWide(data, 2, true, "utf-16-be", 0)
		case data[1] == 0:
			return decodeWide(data, 2, false, "utf-16-le", 0)
		}
	}
	return utf8Text(data)
}

// utf8Text validates data and replaces encoded surrogates (ED A0..BF xx) with U+FFFD.
// 0xED is never a continuation byte, so every 0xED found is a lead byte.
func utf8Text(data []byte) ([]byte, error) {
	if err := checkUTF8(data); err != nil {
		return nil, err
	}
	var out []byte
	for pos := 0; pos < len(data); pos++ {
		if data[pos] == 0xed && data[pos+1] >= 0xa0 {
			if out == nil {
				out = append(make([]byte, 0, len(data)), data[:pos]...)
			}
			out = utf8.AppendRune(out, utf8.RuneError)
			pos += 2
		} else if out != nil {
			out = append(out, data[pos])
		}
	}
	if out == nil {
		return data, nil
	}
	return out, nil
}

func hasPrefix(data []byte, prefix ...byte) bool {
	if len(data) < len(prefix) {
		return false
	}
	for index, b := range prefix {
		if data[index] != b {
			return false
		}
	}
	return true
}

// checkUTF8 validates UTF-8 like CPython's decoder with errors="surrogatepass" and returns
// its UnicodeDecodeError text ("invalid start byte", "invalid continuation byte",
// "unexpected end of data") with the same byte positions.
func checkUTF8(data []byte) error {
	for pos := 0; pos < len(data); {
		c := data[pos]
		if c < 0x80 {
			pos++
			continue
		}
		need, low, high := sequence(c)
		if need == 0 {
			return utf8Error(data, pos, pos+1, "invalid start byte")
		}
		if c == 0xed && pos+1 < len(data) && data[pos+1] >= 0xa0 && data[pos+1] <= 0xbf {
			// An encoded surrogate fails as "invalid continuation byte" on the second byte;
			// the surrogatepass handler then accepts it only as three complete bytes.
			if pos+2 < len(data) && data[pos+2] >= 0x80 && data[pos+2] <= 0xbf {
				pos += 3
				continue
			}
			return utf8Error(data, pos, pos+1, "invalid continuation byte")
		}
		valid := 1
		for valid <= need {
			if pos+valid >= len(data) {
				return utf8Error(data, pos, len(data), "unexpected end of data")
			}
			b := data[pos+valid]
			lo, hi := byte(0x80), byte(0xbf)
			if valid == 1 {
				lo, hi = low, high
			}
			if b < lo || b > hi {
				return utf8Error(data, pos, pos+valid, "invalid continuation byte")
			}
			valid++
		}
		pos += need + 1
	}
	return nil
}

// sequence returns the number of continuation bytes after lead byte c and the allowed range
// of the first continuation byte (RFC 3629, as CPython checks it).
func sequence(c byte) (int, byte, byte) {
	switch {
	case c >= 0xc2 && c <= 0xdf:
		return 1, 0x80, 0xbf
	case c == 0xe0:
		return 2, 0xa0, 0xbf
	case c == 0xed:
		return 2, 0x80, 0x9f
	case c >= 0xe1 && c <= 0xef:
		return 2, 0x80, 0xbf
	case c == 0xf0:
		return 3, 0x90, 0xbf
	case c >= 0xf1 && c <= 0xf3:
		return 3, 0x80, 0xbf
	case c == 0xf4:
		return 3, 0x80, 0x8f
	}
	return 0, 0, 0
}

func utf8Error(data []byte, start, end int, reason string) error {
	return decodeError("utf-8", data, start, end, reason)
}

func decodeError(codec string, data []byte, start, end int, reason string) error {
	if end-start == 1 {
		return apperr.ValueError(fmt.Sprintf("'%s' codec can't decode byte 0x%02x in position %d: %s",
			codec, data[start], start, reason))
	}
	return apperr.ValueError(fmt.Sprintf("'%s' codec can't decode bytes in position %d-%d: %s",
		codec, start, end-1, reason))
}

// decodeWide decodes UTF-16/UTF-32 with surrogatepass. Positions in error texts count from
// the start of the codec input (offset includes a BOM the codec consumed itself).
func decodeWide(data []byte, width int, bigEndian bool, codec string, offset int) ([]byte, error) {
	out := make([]byte, 0, len(data))
	unit := func(pos int) uint32 {
		var v uint32
		for index := 0; index < width; index++ {
			shift := 8 * index
			if bigEndian {
				shift = 8 * (width - 1 - index)
			}
			v |= uint32(data[pos+index]) << shift
		}
		return v
	}
	for pos := 0; pos < len(data); pos += width {
		if pos+width > len(data) {
			full := append(make([]byte, offset), data...)
			return nil, decodeError(codec, full, offset+pos, offset+len(data), "truncated data")
		}
		r := unit(pos)
		if width == 2 && r >= 0xd800 && r < 0xdc00 && pos+4 <= len(data) {
			if next := unit(pos + 2); next >= 0xdc00 && next < 0xe000 {
				r = 0x10000 + (r-0xd800)<<10 + (next - 0xdc00)
				pos += 2
			}
		}
		if r > 0x10ffff {
			full := append(make([]byte, offset), data...)
			return nil, decodeError(codec, full, offset+pos, offset+pos+width, "code point not in range(0x110000)")
		}
		out = appendWTF8(out, r)
	}
	return out, nil
}

// appendWTF8 writes r as UTF-8; a lone surrogate (kept by Python) becomes U+FFFD.
func appendWTF8(dst []byte, r uint32) []byte {
	if r >= 0xd800 && r < 0xe000 {
		r = utf8.RuneError
	}
	return utf8.AppendRune(dst, rune(r))
}
