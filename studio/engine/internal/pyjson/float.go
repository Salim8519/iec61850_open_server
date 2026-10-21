package pyjson

import (
	"math"
	"strconv"
	"strings"
)

// AppendFloat writes f as CPython's json module does: repr(float), and the bare literals
// NaN, Infinity and -Infinity (json.dumps allow_nan=True, also used by Flask).
func AppendFloat(dst []byte, f float64) []byte {
	switch {
	case math.IsNaN(f):
		return append(dst, "NaN"...)
	case math.IsInf(f, 1):
		return append(dst, "Infinity"...)
	case math.IsInf(f, -1):
		return append(dst, "-Infinity"...)
	}
	return append(dst, Repr(f)...)
}

// Repr returns Python's repr of a finite float: the shortest digits that round-trip,
// written positionally when the decimal exponent is between -4 and 15 (always with a
// fractional part, "1.0", "-0.0"), otherwise in scientific form with a signed exponent of
// at least two digits ("1e-05", "1.5e+16").
func Repr(f float64) string {
	scientific := strconv.FormatFloat(f, 'e', -1, 64)
	_, exponentText, _ := strings.Cut(scientific, "e")
	exponent, _ := strconv.Atoi(exponentText)
	if exponent < -4 || exponent >= 16 {
		return scientific
	}
	positional := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(positional, ".") {
		positional += ".0"
	}
	return positional
}
