package contract

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"
)

// rustTrim is Rust's str::trim (Unicode White_Space), which pydantic-core uses.
func rustTrim(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
}

// strAsFloat parses a string like pydantic-core's str_as_float: Rust's f64 grammar on the
// trimmed text, else on the untrimmed text without underscores when none is leading,
// trailing or doubled.
func strAsFloat(s string) (float64, bool) {
	if f, ok := rustFloat(rustTrim(s)); ok {
		return f, true
	}
	if strings.HasPrefix(s, "_") || strings.HasSuffix(s, "_") || strings.Contains(s, "__") {
		return 0, false
	}
	return rustFloat(strings.ReplaceAll(s, "_", ""))
}

// rustFloat is Rust's f64::from_str: sign, then inf/infinity/nan (any case) or decimal
// digits with an optional point and exponent.
func rustFloat(s string) (float64, bool) {
	body := s
	negative := false
	if body != "" && (body[0] == '+' || body[0] == '-') {
		negative = body[0] == '-'
		body = body[1:]
	}
	switch strings.ToLower(body) {
	case "inf", "infinity":
		if negative {
			return math.Inf(-1), true
		}
		return math.Inf(1), true
	case "nan":
		return math.NaN(), true
	}
	if !decimalLiteral(body) {
		return 0, false
	}
	f, err := strconv.ParseFloat(body, 64)
	if err != nil && f == 0 {
		return 0, false
	}
	if negative {
		f = -f
	}
	return f, true
}

func decimalLiteral(s string) bool {
	mantissa, exponent, hasExponent := cutExponent(s)
	if hasExponent {
		if exponent != "" && (exponent[0] == '+' || exponent[0] == '-') {
			exponent = exponent[1:]
		}
		if exponent == "" || !allDigits(exponent) {
			return false
		}
	}
	whole, fraction, _ := strings.Cut(mantissa, ".")
	if whole == "" && fraction == "" {
		return false
	}
	return allDigits(whole) && allDigits(fraction)
}

func cutExponent(s string) (string, string, bool) {
	if index := strings.IndexAny(s, "eE"); index >= 0 {
		return s[:index], s[index+1:], true
	}
	return s, "", false
}

func allDigits(s string) bool {
	for index := 0; index < len(s); index++ {
		if s[index] < '0' || s[index] > '9' {
			return false
		}
	}
	return true
}

// intOutcome is the result of pydantic's string-to-int parsing.
type intOutcome int

const (
	intParsed intOutcome = iota
	intInvalid
	intTooLarge
)

// strAsInt parses a string like pydantic-core's str_as_int: trimmed, at most 4300 bytes,
// an optional sign and ASCII digits (underscores only between digits), optionally followed
// by a point and one or more zeros.
func strAsInt(s string) (*big.Int, intOutcome) {
	t := rustTrim(s)
	if len(t) > 4300 {
		return nil, intTooLarge
	}
	if value, ok := plainInt(t); ok {
		return value, intParsed
	}
	if index := strings.LastIndexByte(t, '.'); index >= 0 {
		zeros := t[index+1:]
		if zeros != "" && strings.Trim(zeros, "0") == "" {
			if value, ok := plainInt(t[:index]); ok {
				return value, intParsed
			}
		}
	}
	return nil, intInvalid
}

func plainInt(s string) (*big.Int, bool) {
	digits := s
	if digits != "" && (digits[0] == '+' || digits[0] == '-') {
		digits = digits[1:]
	}
	if digits == "" || digits[0] == '_' || digits[len(digits)-1] == '_' {
		return nil, false
	}
	for index := 0; index < len(digits); index++ {
		c := digits[index]
		if c == '_' && digits[index-1] >= '0' && digits[index-1] <= '9' &&
			digits[index+1] >= '0' && digits[index+1] <= '9' {
			continue
		}
		if c < '0' || c > '9' {
			return nil, false
		}
	}
	value, ok := new(big.Int).SetString(strings.ReplaceAll(digits, "_", ""), 10)
	if ok && s[0] == '-' {
		value.Neg(value)
	}
	return value, ok
}
