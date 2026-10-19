package pyval

import (
	"math"
	"math/big"
	"strconv"
	"strings"
	"unicode"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl/pyrules"
)

// Str is Python's str(v) (also what an f-string writes) for a decoded JSON value.
func Str(v any) string {
	if text, ok := v.(string); ok {
		return text
	}
	return Repr(v)
}

// Repr is Python's repr(v) for a decoded JSON value.
func Repr(v any) string {
	switch x := v.(type) {
	case nil:
		return "None"
	case bool:
		if x {
			return "True"
		}
		return "False"
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case *big.Int:
		return x.String()
	case float64:
		return FloatStr(x)
	case string:
		return pyrules.Repr(x)
	case []any:
		parts := make([]string, len(x))
		for index, item := range x {
			parts[index] = Repr(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *pyjson.Object:
		parts := make([]string, 0, x.Len())
		for _, key := range x.Keys() {
			value, _ := x.Get(key)
			parts = append(parts, pyrules.Repr(key)+": "+Repr(value))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	}
	return ""
}

// FloatStr is str(float): repr with inf, -inf and nan.
func FloatStr(f float64) string {
	switch {
	case math.IsNaN(f):
		return "nan"
	case math.IsInf(f, 1):
		return "inf"
	case math.IsInf(f, -1):
		return "-inf"
	}
	return pyjson.Repr(f)
}

// Len is len(s) for a str: the number of code points.
func Len(s string) int {
	count := 0
	for range s {
		count++
	}
	return count
}

// Prefix is s[:n] for a str: the first n code points.
func Prefix(s string, n int) string {
	count := 0
	for index := range s {
		if count == n {
			return s[:index]
		}
		count++
	}
	return s
}

// Upper is str.upper(). Go's simple mapping is used, plus the one full (SpecialCasing)
// mapping whose result can matter here: U+FB00 LATIN SMALL LIGATURE FF becomes "FF".
func Upper(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case 0xdf:
			b.WriteString("SS")
		case 0xfb00:
			b.WriteString("FF")
		default:
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}

// Capitalize is str.capitalize() for ASCII setting names: first upper, rest lower.
func Capitalize(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(strings.ToLower(s))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
