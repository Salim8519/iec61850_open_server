package signal

import (
	"math"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

var sourceModes = map[string]bool{"constant": true, "ramp": true, "sine": true, "step": true,
	"noise": true, "manual": true, "csv": true}

// ValidateSource is domain/source.validate_source: the stored candidate spec in Python's key order.
func ValidateSource(item Signal, spec any) (*pyjson.Object, error) {
	if item.FC != "MX" || !(strings.HasPrefix(item.Kind, "FLOAT") || strings.HasPrefix(item.Kind, "INT")) {
		return nil, apperr.ValueError("Automatic sources require numeric MX measurements")
	}
	object, ok := spec.(*pyjson.Object)
	mode, _ := pyval.Get(object, "mode").(string)
	if !ok || !sourceModes[mode] {
		return nil, apperr.ValueError("Unsupported source mode")
	}
	if mode == "csv" {
		return ValidateTrace(item, object)
	}
	floatOf := func(key string, fallback any) (float64, error) {
		value, present := object.Get(key)
		if !present {
			value = fallback
		}
		return pyval.ToFloat(value)
	}
	offset, err := floatOf("offset", int64(0))
	if err != nil {
		return nil, convert(err)
	}
	amplitude, err := floatOf("amplitude", int64(1))
	if err != nil {
		return nil, convert(err)
	}
	period, err := floatOf("period", int64(10))
	if err != nil {
		return nil, convert(err)
	}
	seedValue, present := object.Get("seed")
	if !present {
		seedValue = int64(1)
	}
	seed, err := pyval.ToInt(seedValue)
	if err != nil {
		return nil, convert(err)
	}
	candidate := pyval.Obj("mode", mode, "offset", offset, "amplitude", amplitude, "period", period,
		"seed", seed, "overridden", false)
	for _, number := range []float64{offset, amplitude, period} {
		if math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, apperr.ValueError("Source parameters must be finite numbers")
		}
	}
	if period < 0.1 {
		return nil, apperr.ValueError("Source period must be at least 0.1 seconds")
	}
	return candidate, nil
}

// convert maps a float()/int() failure to the Python exception class Flask maps: ValueError
// is 400; a TypeError is not an apperr and becomes Flask's generic 500.
func convert(err error) error {
	if failure, ok := err.(*pyval.ConvError); ok && !failure.Type {
		return apperr.ValueError(failure.Message)
	}
	return err
}

// Summary is source_summary: the spec without csv_text and points.
func Summary(spec *pyjson.Object) *pyjson.Object {
	summary := pyjson.NewObject()
	for _, key := range spec.Keys() {
		if key == "csv_text" || key == "points" {
			continue
		}
		value, _ := spec.Get(key)
		summary.Set(key, value)
	}
	return summary
}
