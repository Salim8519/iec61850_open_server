package pycompat

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"studio-engine/internal/apperr"
)

// BOM is the UTF-8 byte order mark that the "utf-8-sig" codec removes and writes.
const BOM = "\xef\xbb\xbf"

// DecodeUTF8Sig is bytes.decode("utf-8-sig"): one leading BOM is removed, the rest must be
// strict UTF-8. Error positions count from after the BOM, as in CPython.
func DecodeUTF8Sig(data []byte) (string, error) {
	return DecodeUTF8(trimBOM(data))
}

// EncodeUTF8Sig is str.encode("utf-8-sig"): the BOM followed by the UTF-8 text.
func EncodeUTF8Sig(text string) []byte { return append([]byte(BOM), text...) }

func trimBOM(data []byte) []byte {
	if len(data) >= 3 && string(data[:3]) == BOM {
		return data[3:]
	}
	return data
}

// DecodeUTF8 is bytes.decode("utf-8"); invalid input is a UnicodeDecodeError (a ValueError)
// with CPython's text, e.g. "'utf-8' codec can't decode byte 0xff in position 0: invalid
// start byte".
func DecodeUTF8(data []byte) (string, error) {
	if utf8.Valid(data) {
		return string(data), nil
	}
	start, end, reason := firstUTF8Error(data, 0)
	if end-start == 1 {
		return "", apperr.ValueError(fmt.Sprintf("'utf-8' codec can't decode byte 0x%02x in position %d: %s",
			data[start], start, reason))
	}
	return "", apperr.ValueError(fmt.Sprintf("'utf-8' codec can't decode bytes in position %d-%d: %s",
		start, end-1, reason))
}

// DecodeUTF8Replace is bytes.decode("utf-8", "replace"): every invalid sequence CPython
// reports becomes one U+FFFD.
func DecodeUTF8Replace(data []byte) string {
	return DecodeUTF8With(data, func([]byte) string { return string(utf8.RuneError) })
}

// DecodeUTF8With decodes UTF-8 with a Python error handler: replace receives each
// invalid sequence as CPython's decoder reports it and returns its replacement text.
func DecodeUTF8With(data []byte, replace func(invalid []byte) string) string {
	if utf8.Valid(data) {
		return string(data)
	}
	var out strings.Builder
	for pos := 0; pos < len(data); {
		start, end, _ := firstUTF8Error(data, pos)
		if start < 0 {
			out.Write(data[pos:])
			break
		}
		out.Write(data[pos:start])
		out.WriteString(replace(data[start:end]))
		pos = end
	}
	return out.String()
}

// firstUTF8Error finds the first invalid sequence at or after from, as CPython's decoder
// reports it: [start, end) and the reason; start is -1 when the rest is valid.
func firstUTF8Error(data []byte, from int) (int, int, string) {
	for pos := from; pos < len(data); {
		c := data[pos]
		need, low, high := 0, byte(0x80), byte(0xbf)
		switch {
		case c < 0x80:
			pos++
			continue
		case c < 0xc2 || c > 0xf4:
			return pos, pos + 1, "invalid start byte"
		case c < 0xe0:
			need = 1
		case c < 0xf0:
			need = 2
			if c == 0xe0 {
				low = 0xa0
			} else if c == 0xed {
				high = 0x9f
			}
		default:
			need = 3
			if c == 0xf0 {
				low = 0x90
			} else if c == 0xf4 {
				high = 0x8f
			}
		}
		for i := 1; i <= need; i++ {
			if pos+i >= len(data) {
				return pos, len(data), "unexpected end of data"
			}
			b := data[pos+i]
			if b < low || b > high {
				return pos, pos + i, "invalid continuation byte"
			}
			low, high = 0x80, 0xbf
		}
		pos += need + 1
	}
	return -1, -1, ""
}
