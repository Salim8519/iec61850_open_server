package runtime

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Typed panel values (ADR-0108): range, step and enum checks in plain words.
func TestTypedValueChecks(t *testing.T) {
	inc := pyval.Obj("cdc", "INC", "min", int64(1), "max", int64(17), "step", int64(2))
	apc := pyval.Obj("cdc", "APC", "min", -10.0, "max", 10.0, "step", 0.5)
	enc := pyval.Obj("cdc", "ENC", "labels", pyval.Obj("on", int64(1), "blocked", int64(2), "test", int64(3)))
	cases := []struct {
		name    string
		control *pyjson.Object
		value   any
		want    any
		problem string
	}{
		{"inc ok", inc, int64(5), int64(5), ""},
		{"inc whole float", inc, 7.0, int64(7), ""},
		{"inc above", inc, int64(19), nil, "takes values from 1 up to 17 in steps of 2, not 19"},
		{"inc off step", inc, int64(4), nil, "is between two steps"},
		{"inc fraction", inc, 2.5, nil, "takes a whole number from 1 up to 17 in steps of 2"},
		{"inc bool", inc, true, nil, "takes a whole number"},
		{"apc ok", apc, 2.5, 2.5, ""},
		{"apc below", apc, -10.5, nil, "not -10.5"},
		{"apc off step", apc, 0.2, nil, "between two steps"},
		{"apc text", apc, "5", nil, "takes a number"},
		{"enc ok", enc, "blocked", "blocked", ""},
		{"enc unknown", enc, "off", nil, "takes one of: on, blocked, test"},
		{"enc number", enc, int64(1), nil, "takes one of"},
		{"enc no labels", pyval.Obj("cdc", "ENC"), "on", nil, "has no list of values"},
		{"bsc raise", pyval.Obj("cdc", "BSC"), "higher", "higher", ""},
		{"isc lower", pyval.Obj("cdc", "ISC"), "lower", "lower", ""},
		{"bsc stop", pyval.Obj("cdc", "BSC"), "stop", nil, "raise/lower command"},
		{"spc bool", pyval.Obj("cdc", "SPC"), true, true, ""},
		{"dpc number", pyval.Obj("cdc", "DPC"), int64(1), nil, "is sent as true or false"},
		{"bac number", pyval.Obj("cdc", "BAC"), 1.0, nil, "raise/lower command"},
		{"bac raise", pyval.Obj("cdc", "BAC"), "higher", "higher", ""},
		{"unknown class", pyval.Obj("cdc", "LTRK"), 1.0, nil, "cannot command"},
		{"apc no bounds", pyval.Obj("cdc", "APC"), 1e6, 1e6, ""},
	}
	for _, test := range cases {
		got, err := typedValue(test.control, test.value, "Tap")
		if test.problem == "" {
			if err != nil || !pyval.Equal(got, test.want) {
				t.Errorf("%s: %v %v, want %v", test.name, got, err, test.want)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), test.problem) || !strings.HasPrefix(err.Error(), "Tap ") {
			t.Errorf("%s: %v, want %q", test.name, err, test.problem)
		}
	}
}
