package pytext

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

// ParseInt is int(text) for a str: surrounding whitespace, a sign, decimal digits of any
// script and single underscores between digits. It returns int64 or *big.Int, or a
// ValueError text exactly as CPython words it.
func ParseInt(text string) (any, error) {
	digits, ok := asciiNumber(text)
	if ok {
		ok = validInt(digits)
	}
	if !ok {
		return nil, &ValueError{"invalid literal for int() with base 10: " + Head(ReprString(text), 200)}
	}
	digits = strings.ReplaceAll(digits, "_", "")
	if small, err := strconv.ParseInt(digits, 10, 64); err == nil {
		return small, nil
	}
	large, _ := new(big.Int).SetString(digits, 10)
	return large, nil
}

// ParseFloat is float(text) for a str: whitespace, sign, digits of any script, single
// underscores between digits, inf/infinity/nan in any case. Hexadecimal is refused.
func ParseFloat(text string) (float64, error) {
	fail := &ValueError{"could not convert string to float: " + ReprString(text)}
	number, ok := asciiNumber(text)
	if !ok {
		return 0, fail
	}
	body := strings.TrimLeft(number, "+-")
	if len(number)-len(body) > 1 {
		return 0, fail
	}
	switch strings.ToLower(body) {
	case "nan":
		return math.NaN(), nil // Go's ParseFloat refuses a signed NaN; Python accepts "-nan"
	case "inf", "infinity":
		value, _ := strconv.ParseFloat(number, 64)
		return value, nil
	}
	if !validFloat(body) {
		return 0, fail
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(number, "_", ""), 64)
	if err != nil && value == 0 && !strings.ContainsAny(body, "123456789") {
		return 0, fail
	}
	return value, nil // overflow gives ±Inf like Python's float("1e999")
}

// asciiNumber strips number whitespace and turns digits of other scripts into ASCII, as
// CPython's _PyUnicode_TransformDecimalAndSpaceToASCII does before parsing.
func asciiNumber(text string) (string, bool) {
	text = strings.TrimFunc(text, numberSpace)
	var out strings.Builder
	for _, r := range text {
		switch {
		case r < 0x80:
			out.WriteRune(r)
		case unicode.Is(unicode.Nd, r):
			out.WriteByte(byte('0' + digitValue(r)))
		default:
			return "", false
		}
	}
	return out.String(), true
}

// digitValue: Unicode Nd digits come in runs of ten starting at a zero (some runs touch,
// like the mathematical digits), so the value is the distance from the run start mod 10.
func digitValue(r rune) int {
	start := r
	for unicode.Is(unicode.Nd, start-1) {
		start--
	}
	return int(r-start) % 10
}

func validInt(text string) bool {
	if len(text) > 0 && (text[0] == '+' || text[0] == '-') {
		text = text[1:]
	}
	return digitRun(text)
}

// digitRun: one or more digits with single underscores only between digits.
func digitRun(text string) bool {
	if text == "" || text[0] == '_' || text[len(text)-1] == '_' {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if c == '_' {
			if text[index+1] == '_' {
				return false
			}
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validFloat(body string) bool {
	mantissa, exponent, hasExponent := strings.Cut(strings.ToLower(body), "e")
	if hasExponent {
		exponent = strings.TrimPrefix(strings.TrimPrefix(exponent, "+"), "-")
		if !digitRun(exponent) || strings.Contains(exponent, "e") {
			return false
		}
	}
	whole, fraction, hasPoint := strings.Cut(mantissa, ".")
	if !hasPoint {
		return digitRun(whole)
	}
	if whole == "" && fraction == "" {
		return false
	}
	return (whole == "" || digitRun(whole)) && (fraction == "" || digitRun(fraction))
}

// ValueError carries a Python ValueError text from a parser.
type ValueError struct{ Text string }

func (e *ValueError) Error() string { return e.Text }
