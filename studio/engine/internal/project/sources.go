package project

import (
	"math"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/csvtrace"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// signalKey is the (IED id, reference) pair Python uses as a dict key.
type signalKey struct{ ied, ref string }

// signalIndex is {(ied["id"], signal["ref"]): signal}; a repeated key keeps the last signal.
func signalIndex(ieds []*pyjson.Object) map[signalKey]*pyjson.Object {
	index := map[signalKey]*pyjson.Object{}
	for _, ied := range ieds {
		for _, item := range items(pyval.Lookup(ied, "signals", nil)) {
			signal := object(item)
			index[signalKey{text(ied, "id"), text(signal, "ref")}] = signal
		}
	}
	return index
}

// lookupKey hashes (ied_id, ref) like a Python tuple key: lists and dicts raise TypeError,
// other non-str values never match.
func lookupKey(iedID, ref any) (signalKey, bool, error) {
	for _, part := range []any{iedID, ref} {
		if err := pyval.CheckHashable(part); err != nil {
			return signalKey{}, false, err
		}
	}
	ied, iedIsText := iedID.(string)
	reference, refIsText := ref.(string)
	return signalKey{ied, reference}, iedIsText && refIsText, nil
}

// ValidateSources is validate_sources: automatic numeric sources of the project.
func ValidateSources(sources any, ieds []*pyjson.Object) ([]any, error) {
	list, isList := sources.([]any)
	if !isList || len(list) > 50000 {
		return nil, apperr.New("Invalid source definitions")
	}
	signals := signalIndex(ieds)
	result, seen := []any{}, map[signalKey]bool{}
	for _, entry := range list {
		item, isDict := entry.(*pyjson.Object)
		if !isDict {
			return nil, apperr.New("Invalid source definition")
		}
		iedID, ref := pyval.Lookup(item, "ied_id", nil), pyval.Lookup(item, "ref", nil)
		key, textual, err := lookupKey(iedID, ref)
		if err != nil {
			return nil, err
		}
		signal := signals[key]
		if !textual || signal == nil || seen[key] {
			return nil, apperr.New("Source references an unknown or duplicate signal")
		}
		seen[key] = true
		spec, err := ValidateSource(signal, pyval.Lookup(item, "spec", nil))
		if err != nil {
			return nil, err
		}
		result = append(result, pyjson.NewObject().Set("ied_id", iedID).Set("ref", ref).Set("spec", spec))
	}
	return result, nil
}

// ValidateSource is domain/source.validate_source: one generator or CSV source spec.
func ValidateSource(signal *pyjson.Object, spec any) (*pyjson.Object, error) {
	kind := text(signal, "kind")
	if text(signal, "fc") != "MX" || !(strings.HasPrefix(kind, "FLOAT") || strings.HasPrefix(kind, "INT")) {
		return nil, apperr.ValueError("Automatic sources require numeric MX measurements")
	}
	given, isDict := spec.(*pyjson.Object)
	if !isDict {
		return nil, apperr.ValueError("Unsupported source mode")
	}
	mode := pyval.Lookup(given, "mode", nil)
	known, err := pyval.InStrings(mode, "constant", "ramp", "sine", "step", "noise", "manual", "csv")
	if err != nil {
		return nil, err
	}
	if !known {
		return nil, apperr.ValueError("Unsupported source mode")
	}
	if mode == "csv" {
		return csvtrace.ValidateTrace(signal, given)
	}
	candidate := pyjson.NewObject().Set("mode", mode)
	for _, parameter := range []struct {
		name     string
		fallback any
	}{{"offset", int64(0)}, {"amplitude", int64(1)}, {"period", int64(10)}} {
		number, err := pyval.Float(pyval.Lookup(given, parameter.name, parameter.fallback))
		if err != nil {
			return nil, err
		}
		candidate.Set(parameter.name, number)
	}
	seed, err := pyval.Int(pyval.Lookup(given, "seed", int64(1)))
	if err != nil {
		return nil, err
	}
	candidate.Set("seed", seed).Set("overridden", false)
	for _, name := range []string{"offset", "amplitude", "period"} {
		number, _ := candidate.Get(name)
		if f := number.(float64); math.IsInf(f, 0) || math.IsNaN(f) {
			return nil, apperr.ValueError("Source parameters must be finite numbers")
		}
	}
	if period, _ := candidate.Get("period"); period.(float64) < 0.1 {
		return nil, apperr.ValueError("Source period must be at least 0.1 seconds")
	}
	return candidate, nil
}

// SourceSummary is source_summary: a source spec without its CSV text and points.
func SourceSummary(spec *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	for _, key := range spec.Keys() {
		if key != "csv_text" && key != "points" {
			value, _ := spec.Get(key)
			result.Set(key, value)
		}
	}
	return result
}
