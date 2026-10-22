package typedcmd

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func TestWireValues(t *testing.T) {
	labels := pyjson.NewObject().Set("on", int64(1)).Set("test", int64(3))
	cases := []struct {
		control Control
		value   any
		current any
		kind    string
		wire    any
		err     string
	}{
		{Control{ValueKind: "bool"}, true, nil, "bool", true, ""},
		{Control{ValueKind: "bool"}, int64(1), nil, "", nil, "Control value must be boolean"},
		{Control{ValueKind: "enum", Labels: labels}, "test", nil, "enum", int64(3), ""},
		{Control{ValueKind: "enum", Labels: labels}, int64(1), nil, "enum", int64(1), ""},
		{Control{ValueKind: "enum", Labels: labels, Ref: "X"}, "off", nil, "", nil, "Pick one of the listed values for X"},
		{Control{ValueKind: "int", Min: int64(0), Max: int64(10)}, int64(7), nil, "int", int64(7), ""},
		{Control{ValueKind: "int", Min: int64(0), Max: int64(10)}, 7.0, nil, "int", int64(7), ""},
		{Control{ValueKind: "int", Min: int64(0), Max: int64(10)}, int64(11), nil, "", nil, "between 0 and 10"},
		{Control{ValueKind: "int"}, 1.5, nil, "", nil, "whole number"},
		{Control{ValueKind: "int", Min: int64(1), Max: int64(17)}, "higher", int64(5), "int", int64(6), ""},
		{Control{ValueKind: "int", Min: int64(1), Max: int64(17)}, "lower", int64(1), "", nil, "between 1 and 17"},
		{Control{ValueKind: "int"}, "higher", nil, "", nil, "present position is unknown"},
		{Control{ValueKind: "float", Min: 0.0, Max: 100.0}, 42.5, nil, "float", 42.5, ""},
		{Control{ValueKind: "float", Min: 0.0, Max: 100.0}, int64(120), nil, "", nil, "between 0 and 100"},
		{Control{ValueKind: "step"}, "higher", nil, "step", "higher", ""},
		{Control{ValueKind: "step"}, int64(1), nil, "", nil, "higher, lower or stop"},
	}
	for _, c := range cases {
		kind, wire, err := Wire(c.control, c.value, c.current)
		if c.err != "" {
			if err == nil || !strings.Contains(err.Error(), c.err) {
				t.Errorf("%+v %v: error %v, want %q", c.control, c.value, err, c.err)
			}
			continue
		}
		if err != nil || kind != c.kind || wire != c.wire {
			t.Errorf("%+v %v: %q %v %v", c.control, c.value, kind, wire, err)
		}
	}
}

func TestOptionsAndOutcome(t *testing.T) {
	options, err := ParseOptions(pyjson.NewObject().Set("synchro_check", true).Set("orcat", int64(7)))
	if err != nil || !options.SynchroCheck || options.OrCat != 7 {
		t.Fatalf("%+v %v", options, err)
	}
	data, _ := pyjson.Dumps(Options{}.Fields(pyjson.NewObject()))
	if string(data) != `{"or_ident": "ScadaBench", "or_cat": 0, "interlock_check": false, "synchro_check": false, "test": false}` {
		t.Fatalf("fields %s", data)
	}
	if _, err := ParseOptions(pyjson.NewObject().Set("orident", strings.Repeat("x", 65))); err == nil {
		t.Fatal("orIdent is at most 64 octets")
	}
	if err := Outcome(pyjson.NewObject().Set("ok", false).Set("error", "Operate refused by the device")); err == nil ||
		err.Error() != "Operate refused by the device" {
		t.Fatalf("outcome: %v", err)
	}
	if err := SelectExpired(30000); err.Error() != "Select expired after 30 s. Press Select again." {
		t.Fatal(err)
	}
	if ModelName(4) != "sbo-enhanced" || ModelName(0) != "status-only" {
		t.Fatal("model names")
	}
}
