// Package sld builds a single-line diagram from an SCL substation topology exactly as
// domain/sld_layout.py, sld_auto_layout.py and sld_bindings.py do: the same node ids,
// positions (IEEE doubles in Python's evaluation order), wires, bindings and diagnostics.
package sld

import "math"

// pyRound is round(x) for a float: half to even, as an int.
func pyRound(x float64) int64 { return int64(math.RoundToEven(x)) }

// floorDiv is Python's float // (float_floor_div in floatobject.c).
func floorDiv(vx, wx float64) float64 {
	mod := math.Mod(vx, wx)
	div := (vx - mod) / wx
	if mod != 0 {
		if (wx < 0) != (mod < 0) {
			div -= 1.0
		}
	}
	if div == 0 {
		return math.Copysign(0, vx/wx)
	}
	floor := math.Floor(div)
	if div-floor > 0.5 {
		floor += 1.0
	}
	return floor
}

// pySum is sum(values) of floats in CPython 3.12 (Neumaier compensated summation).
func pySum(values []float64) float64 {
	result, compensation := 0.0, 0.0
	for _, x := range values {
		t := result + x
		if math.Abs(result) >= math.Abs(x) {
			compensation += (result - t) + x
		} else {
			compensation += (x - t) + result
		}
		result = t
	}
	if compensation != 0 && !math.IsInf(compensation, 0) && !math.IsNaN(compensation) {
		result += compensation
	}
	return result
}

// pyMin is min(a, b): b only when b < a (NaN never wins a comparison).
func pyMin(a, b float64) float64 {
	if b < a {
		return b
	}
	return a
}

// ordered is a Python dict of str -> []string with insertion order.
type ordered struct {
	keys   []string
	values map[string][]string
}

func newOrdered() *ordered { return &ordered{values: map[string][]string{}} }

func (o *ordered) add(key, value string) {
	list, seen := o.values[key]
	if !seen {
		o.keys = append(o.keys, key)
	}
	for _, existing := range list {
		if existing == value {
			return
		}
	}
	o.values[key] = append(list, value)
}
