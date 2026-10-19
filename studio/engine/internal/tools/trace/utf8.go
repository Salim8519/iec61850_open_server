package trace

import (
	"bytes"
	"fmt"

	"studio-engine/internal/apperr"
)

// DecodeUTF8Sig is bytes.decode("utf-8-sig"): one leading BOM removed, and on invalid
// input CPython's UnicodeDecodeError text (a ValueError, so the route answers 400).
func DecodeUTF8Sig(payload []byte) (string, error) {
	payload = bytes.TrimPrefix(payload, []byte{0xef, 0xbb, 0xbf})
	if start, end, reason, bad := invalidUTF8(payload); bad {
		if end-start == 1 {
			return "", apperr.ValueError(fmt.Sprintf("'utf-8' codec can't decode byte 0x%02x in position %d: %s",
				payload[start], start, reason))
		}
		return "", apperr.ValueError(fmt.Sprintf("'utf-8' codec can't decode bytes in position %d-%d: %s",
			start, end-1, reason))
	}
	return string(payload), nil
}

func continuation(b byte) bool { return b >= 0x80 && b <= 0xbf }

// invalidUTF8 returns the first error range as CPython's decoder reports it.
func invalidUTF8(data []byte) (int, int, string, bool) {
	const (
		badStart = "invalid start byte"
		badNext  = "invalid continuation byte"
		short    = "unexpected end of data"
	)
	for i := 0; i < len(data); {
		c := data[i]
		var size int
		var low, high byte = 0x80, 0xbf // allowed range of the second byte
		switch {
		case c < 0x80:
			i++
			continue
		case c >= 0xc2 && c <= 0xdf:
			size = 2
		case c >= 0xe0 && c <= 0xef:
			size = 3
			if c == 0xe0 {
				low = 0xa0
			} else if c == 0xed {
				high = 0x9f
			}
		case c >= 0xf0 && c <= 0xf4:
			size = 4
			if c == 0xf0 {
				low = 0x90
			} else if c == 0xf4 {
				high = 0x8f
			}
		default:
			return i, i + 1, badStart, true
		}
		for k := 1; k < size; k++ {
			if i+k >= len(data) {
				return i, len(data), short, true
			}
			b := data[i+k]
			if k == 1 && (b < low || b > high) || k > 1 && !continuation(b) {
				return i, i + k, badNext, true
			}
		}
		i += size
	}
	return 0, 0, "", false
}
