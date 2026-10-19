package pyrules

import (
	"fmt"

	"studio-engine/internal/apperr"
)

// DecodeUTF8Sig is bytes.decode("utf-8-sig"): the text without a leading byte order mark,
// or Python's UnicodeDecodeError (a ValueError) with CPython's exact message. Positions
// count from after a stripped mark, as CPython's utf_8_sig decoder does.
func DecodeUTF8Sig(data []byte) (string, error) {
	if len(data) >= 3 && data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf {
		data = data[3:]
	} else if len(data) < 3 && len(data) > 0 && isBOMPrefix(data) {
		return "", decodeError(data, 0, len(data), "unexpected end of data")
	}
	for i := 0; i < len(data); {
		c := data[i]
		if c < 0x80 {
			i++
			continue
		}
		size, low, high := 0, byte(0x80), byte(0xbf)
		switch {
		case c >= 0xc2 && c <= 0xdf:
			size = 2
		case c == 0xe0:
			size, low = 3, 0xa0
		case c >= 0xe1 && c <= 0xec || c == 0xee || c == 0xef:
			size = 3
		case c == 0xed:
			size, high = 3, 0x9f
		case c == 0xf0:
			size, low = 4, 0x90
		case c >= 0xf1 && c <= 0xf3:
			size = 4
		case c == 0xf4:
			size, high = 4, 0x8f
		default:
			return "", decodeError(data, i, i+1, "invalid start byte")
		}
		for k := 1; k < size; k++ {
			if i+k >= len(data) {
				return "", decodeError(data, i, len(data), "unexpected end of data")
			}
			b := data[i+k]
			lo, hi := byte(0x80), byte(0xbf)
			if k == 1 {
				lo, hi = low, high
			}
			if b < lo || b > hi {
				return "", decodeError(data, i, i+k, "invalid continuation byte")
			}
		}
		i += size
	}
	return string(data), nil
}

func isBOMPrefix(data []byte) bool {
	bom := []byte{0xef, 0xbb, 0xbf}
	for i, c := range data {
		if c != bom[i] {
			return false
		}
	}
	return true
}

func decodeError(data []byte, start, end int, reason string) error {
	if end-start == 1 {
		return apperr.ValueError(fmt.Sprintf("'utf-8' codec can't decode byte 0x%02x in position %d: %s", data[start], start, reason))
	}
	return apperr.ValueError(fmt.Sprintf("'utf-8' codec can't decode bytes in position %d-%d: %s", start, end-1, reason))
}
