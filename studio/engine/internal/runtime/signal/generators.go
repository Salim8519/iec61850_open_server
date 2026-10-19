package signal

import (
	"fmt"
	"math"
	"math/big"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// ValueAt is simulation/generators.value_at with the same float arithmetic, step by step.
// Every product is wrapped in float64(...) so Go never fuses it into a multiply-add.
func ValueAt(spec *pyjson.Object, elapsed float64) (any, error) {
	start, _ := pyval.Float(pyval.Get(spec, "start_elapsed"))
	elapsed = math.Max(0, elapsed-start)
	if elapsed == 0 {
		elapsed = 0 // max(0, x) returns the int 0 in Python; arithmetic below is the same
	}
	kind, present := spec.Get("mode")
	if !present {
		kind = "constant"
	}
	if kind == "csv" {
		return TraceValue(spec, elapsed), nil
	}
	number := func(key string, fallback float64) (float64, error) {
		value, present := spec.Get(key)
		if !present {
			return fallback, nil
		}
		return pyval.ToFloat(value)
	}
	offset, err := number("offset", 0)
	if err != nil {
		return nil, err
	}
	amplitude, err := number("amplitude", 1)
	if err != nil {
		return nil, err
	}
	period, err := number("period", 10)
	if err != nil {
		return nil, err
	}
	period = math.Max(0.1, period)
	switch kind {
	case "constant":
		return offset, nil
	case "ramp":
		return offset + float64(amplitude*(pyMod(elapsed, period)/period)), nil
	case "sine":
		angle := float64(float64(2*math.Pi)*elapsed) / period
		return offset + float64(amplitude*math.Sin(angle)), nil
	case "step":
		if pyMod(elapsed, period) >= period/2 {
			return offset + amplitude, nil
		}
		return offset + 0, nil
	case "noise":
		seedValue, present := spec.Get("seed")
		if !present {
			seedValue = int64(1)
		}
		seed, err := pyval.ToInt(seedValue)
		if err != nil {
			return nil, err
		}
		tenths, err := pyval.ToInt(float64(elapsed * 10))
		if err != nil {
			return nil, err
		}
		key := new(big.Int).Add(bigOf(seed), bigOf(tenths))
		return offset + newPyRandom(key).uniform(-amplitude, amplitude), nil
	}
	return nil, apperr.ValueError(fmt.Sprintf("Unknown generator %s", pyval.Repr(kind)))
}

func bigOf(value any) *big.Int {
	switch typed := value.(type) {
	case int64:
		return big.NewInt(typed)
	case *big.Int:
		return typed
	}
	return big.NewInt(0)
}
