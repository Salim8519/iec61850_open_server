package pycompat

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
)

// Round is round(x) for a float: the nearest integer, halves to even, as int64 when it
// fits, otherwise *big.Int. NaN is a ValueError and infinities an OverflowError.
func Round(x float64) (any, error) {
	switch {
	case math.IsNaN(x):
		return nil, apperr.ValueError("cannot convert float NaN to integer")
	case math.IsInf(x, 0):
		return nil, overflow("cannot convert float infinity to integer")
	}
	rounded := math.RoundToEven(x)
	if rounded >= -(1<<63) && rounded < 1<<63 {
		return int64(rounded), nil
	}
	value, _ := big.NewFloat(rounded).Int(nil)
	return value, nil
}

// Python's limits for round(x, ndigits): beyond them x is returned or becomes ±0.0.
const (
	roundDigitsMax = 323
	roundDigitsMin = -308
)

// RoundDigits is round(x, ndigits) for a float: correctly rounded to ndigits decimal
// places (negative ndigits round to tens, hundreds...), exact halves to even, the sign of
// zero kept. NaN and infinities are returned unchanged.
func RoundDigits(x float64, ndigits int) (float64, error) {
	if math.IsNaN(x) || math.IsInf(x, 0) || ndigits > roundDigitsMax {
		return x, nil
	}
	if ndigits < roundDigitsMin {
		return 0.0 * x, nil
	}
	var text string
	if ndigits >= 0 {
		text = strconv.FormatFloat(x, 'f', ndigits, 64)
	} else {
		text = roundIntegerPart(x, -ndigits)
	}
	rounded, err := strconv.ParseFloat(text, 64)
	if err != nil && math.IsInf(rounded, 0) {
		return 0, overflow("rounded value too large to represent")
	}
	if rounded == 0 && math.Signbit(x) {
		rounded = math.Copysign(0, -1)
	}
	return rounded, nil
}

// roundIntegerPart rounds |x| to a multiple of 10**places, halves to even, using the exact
// integer part and whether a fraction remains, and returns it as "[-]digits e places".
func roundIntegerPart(x float64, places int) string {
	whole, _ := big.NewFloat(math.Abs(x)).Int(nil)
	fraction := math.Abs(x) != math.Trunc(math.Abs(x))
	digits := whole.String()
	if len(digits) <= places {
		digits = strings.Repeat("0", places-len(digits)+1) + digits
	}
	keep, dropped := digits[:len(digits)-places], digits[len(digits)-places:]
	up := false
	switch {
	case dropped[0] > '5':
		up = true
	case dropped[0] == '5':
		rest := strings.TrimRight(dropped[1:], "0") != "" || fraction
		up = rest || (keep[len(keep)-1]-'0')%2 == 1
	}
	kept, _ := new(big.Int).SetString(keep, 10)
	if up {
		kept.Add(kept, big.NewInt(1))
	}
	sign := ""
	if math.Signbit(x) {
		sign = "-"
	}
	return sign + kept.String() + "e" + strconv.Itoa(places)
}
