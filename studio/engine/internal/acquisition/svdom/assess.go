package svdom

import (
	"math"
	"math/big"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Assess returns {state, problems} of one stream from a native `sv_rx` report.
// previous is the stream's last report (nil when there is none); a receiving stream
// with any problem reads "problem".
func Assess(watch, event, previous *pyjson.Object) *pyjson.Object {
	state := pyv.GetOr(event, "state", "waiting")
	problems := []any{}
	if pyv.Truthy(pyv.Get(event, "asdus")) {
		if !pyv.Equal(pyv.Get(event, "conf_rev"), pyv.Get(watch, "conf_rev")) {
			problems = append(problems, "conf_rev_mismatch")
		}
		dataSize := pyv.Get(event, "data_size")
		if pyv.Equal(pyv.Get(watch, "layout"), "9-2LE") && dataSize != nil && !pyv.Equal(dataSize, int64(64)) {
			problems = append(problems, "dataset_mismatch")
		}
		rate, expected := orZero(pyv.Get(event, "rate")), pyv.Get(watch, "expected_rate")
		if pyv.Equal(state, "receiving") && pyv.Truthy(expected) && pyv.Truthy(rate) &&
			beyondTolerance(rate, expected) {
			problems = append(problems, "rate_mismatch")
		}
		if pyv.Equal(pyv.Get(event, "smp_synch"), int64(0)) {
			problems = append(problems, "not_synchronised")
		}
		if greater(orZero(pyv.Get(event, "lost")), orZero(pyv.Get(previous, "lost"))) {
			problems = append(problems, "sample_loss")
		}
		if greater(orZero(pyv.Get(event, "invalid_quality")), orZero(pyv.Get(previous, "invalid_quality"))) {
			problems = append(problems, "invalid_quality")
		}
		if pyv.Truthy(pyv.Get(event, "simulate")) {
			problems = append(problems, "simulation")
		}
	}
	// Problems are appended in PROBLEM_ORDER already, so Python's sort leaves them as they are.
	if pyv.Equal(state, "receiving") && len(problems) > 0 {
		state = "problem"
	}
	return pyv.O("state", state, "problems", problems)
}

// orZero is `value or 0`.
func orZero(value any) any {
	if pyv.Truthy(value) {
		return value
	}
	return int64(0)
}

// beyondTolerance is abs(rate - expected) / expected > RATE_TOLERANCE with Python's
// arithmetic: exact integers divided with correct rounding, otherwise floats.
func beyondTolerance(rate, expected any) bool {
	x, xInt := bigInt(rate)
	y, yInt := bigInt(expected)
	var ratio float64
	if xInt && yInt {
		diff := new(big.Int).Sub(x, y)
		ratio, _ = new(big.Rat).SetFrac(diff.Abs(diff), y).Float64()
	} else {
		fr, okRate := toFloat(rate)
		fe, okExpected := toFloat(expected)
		if !okRate || !okExpected {
			return false
		}
		ratio = math.Abs(fr-fe) / fe
	}
	return ratio > RateTolerance
}
