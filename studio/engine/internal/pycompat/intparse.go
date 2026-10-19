package pycompat

import (
	"fmt"
	"math/big"
	"unicode"
	"unicode/utf8"

	"studio-engine/internal/apperr"
)

// MaxIntDigits is sys.get_int_max_str_digits() (CPython default 4300).
const MaxIntDigits = 4300

// ParseInt is int(text) for a str, base 10: optional surrounding whitespace (Unicode
// whitespace, but not U+001C..U+001F), one optional sign, decimal digits of any script,
// single underscores between digits. It returns int64 when the value fits, else *big.Int.
// Failures are ValueError with CPython's message.
func ParseInt(text string) (any, error) {
	value, digits, ok := parseASCIIInt(transformDecimalAndSpace(text))
	if !ok {
		return nil, invalidInt(text)
	}
	if digits > MaxIntDigits {
		return nil, apperr.ValueError(fmt.Sprintf("Exceeds the limit (%d digits) for integer string "+
			"conversion: value has %d digits; use sys.set_int_max_str_digits() to increase the limit",
			MaxIntDigits, digits))
	}
	if value.IsInt64() {
		return value.Int64(), nil
	}
	return value, nil
}

func invalidInt(text string) error {
	return apperr.ValueError("invalid literal for int() with base 10: " + TruncatedRepr(text))
}

// transformDecimalAndSpace is _PyUnicode_TransformDecimalAndSpaceToASCII: ASCII stays,
// Unicode whitespace becomes ' ', decimal digits become '0'..'9'; anything else ends the
// text with a '?' (as CPython does), which can never parse.
func transformDecimalAndSpace(text string) []byte {
	out := make([]byte, 0, len(text))
	for index, r := range text {
		switch {
		case r == utf8.RuneError && isInvalidAt(text, index):
			return append(out, '?')
		case r < 0x7f:
			out = append(out, byte(r))
		case IsSpace(r):
			out = append(out, ' ')
		default:
			digit := DecimalValue(r)
			if digit < 0 {
				return append(out, '?')
			}
			out = append(out, byte('0'+digit))
		}
	}
	return out
}

func isInvalidAt(text string, index int) bool {
	_, size := utf8.DecodeRuneInString(text[index:])
	return size <= 1
}

// parseASCIIInt is PyLong_FromString for base 10 over the transformed text, requiring
// the whole text to be consumed. digits counts the digits, for the 4300-digit limit that
// CPython applies only to otherwise valid text.
func parseASCIIInt(text []byte) (*big.Int, int, bool) {
	at := func(i int) byte {
		if i < len(text) {
			return text[i]
		}
		return 0
	}
	pos := 0
	for at(pos) != 0 && IsASCIISpace(at(pos)) {
		pos++
	}
	negative := false
	if at(pos) == '+' || at(pos) == '-' {
		negative = at(pos) == '-'
		pos++
	}
	if at(pos) == '_' {
		return nil, 0, false
	}
	start, digits, previous := pos, 0, byte(0)
	var number []byte
	for ; isDigitByte(at(pos)) || at(pos) == '_'; pos++ {
		if at(pos) == '_' && previous == '_' {
			return nil, 0, false
		}
		if at(pos) != '_' {
			number = append(number, at(pos))
			digits++
		}
		previous = at(pos)
	}
	if previous == '_' || pos == start {
		return nil, 0, false
	}
	for at(pos) != 0 && IsASCIISpace(at(pos)) {
		pos++
	}
	if pos != len(text) {
		return nil, digits, false
	}
	value, _ := new(big.Int).SetString(string(number), 10)
	if negative {
		value.Neg(value)
	}
	return value, digits, true
}

func isDigitByte(b byte) bool { return b >= '0' && b <= '9' }

// DecimalValue is unicodedata.decimal(r, -1): the value of a Unicode decimal digit
// (category Nd), whose runs of ten always start at a zero.
func DecimalValue(r rune) int {
	if r >= '0' && r <= '9' {
		return int(r - '0')
	}
	if !unicode.Is(unicode.Nd, r) {
		return -1
	}
	for _, span := range unicode.Nd.R16 {
		if r >= rune(span.Lo) && r <= rune(span.Hi) {
			return int(r-rune(span.Lo)) % 10
		}
	}
	for _, span := range unicode.Nd.R32 {
		if r >= rune(span.Lo) && r <= rune(span.Hi) {
			return int(r-rune(span.Lo)) % 10
		}
	}
	return -1
}
