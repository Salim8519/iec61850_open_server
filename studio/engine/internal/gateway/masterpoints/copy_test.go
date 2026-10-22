package masterpoints

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func gatewayPoint(kind string, address int64, description string) *pyjson.Object {
	return obj("id", "FEED1:FEED1LD0/XCBR1.Pos.stVal:"+kind, "kind", kind, "ied", "FEED1", "ref", "FEED1LD0/XCBR1.Pos.stVal",
		"address", address, "scale", 1.0, "description", description, "ied_name", "FEED1")
}

// Only the address, the master's type and the name are copied; the IED is never linked.
func TestCopyFromGatewayMapsTypesAndKeepsNoIEDLink(t *testing.T) {
	source := []any{gatewayPoint("double", 20001, "Feeder 1 breaker"), gatewayPoint("command_double", 50001, ""),
		gatewayPoint("integer", 40001, "Feeder 1 current")}
	results, notes := FromGateway("iec104", source, nil, nil)
	if !OK(results) || len(notes) != 0 || len(results) != 3 {
		t.Fatalf("copy: %s %v", jsonText(t, PlanJSON(results)), notes)
	}
	first := jsonText(t, results[0].Point)
	if first != `{"id":"m1","name":"FEED1 Feeder 1 breaker","type":"double","address":20001,"scale":1.0,"offset":0.0,"unit":"",`+
		`"from":{"protocol":"iec104","address":20001,"kind":"double"}}` || strings.Contains(first, "FEED1LD0") {
		t.Fatalf("copied point: %s", first)
	}
	if text(results[1].Point, "type") != "command_double" || text(results[1].Point, "name") != "FEED1 XCBR1 Pos" ||
		text(results[2].Point, "type") != "scaled" {
		t.Fatalf("types and names: %s", jsonText(t, PlanJSON(results)))
	}
	current := []any{results[0].Point, obj("id", "m9", "type", "single", "address", int64(40001), "name", "Mine")}
	again, notes := FromGateway("iec104", source, current, nil)
	if len(again) != 1 || text(again[0].Point, "type") != "command_double" || len(notes) != 2 ||
		notes[0] != "1 gateway points are already in your list." || !strings.Contains(notes[1], `IOA 40001 is already used by "Mine"`) {
		t.Fatalf("copy again adds only new points: %s %v", jsonText(t, PlanJSON(again)), notes)
	}
	dnp3, _ := FromGateway("dnp3", []any{gatewayPoint("single", 0, "a"), gatewayPoint("command_double", 0, "b")}, nil, nil)
	modbus, _ := FromGateway("modbus", []any{gatewayPoint("float", 1000, "c"), gatewayPoint("command_single", 0, "d")}, nil, map[string]string{"FEED1": "Feeder one"})
	if text(dnp3[0].Point, "type") != "binary" || text(dnp3[1].Point, "code") != "trip_close" ||
		text(modbus[0].Point, "type") != "holding_float32" || text(modbus[1].Point, "command") != "coil" ||
		text(modbus[1].Point, "name") != "Feeder one d" {
		t.Fatalf("dnp3 %s modbus %s", jsonText(t, PlanJSON(dnp3)), jsonText(t, PlanJSON(modbus)))
	}
}

func checked(t *testing.T, item *pyjson.Object, mapped Mapped) (any, string) {
	t.Helper()
	answer := CheckAgainstIED("iec104", item, mapped)
	if answer == nil {
		return "none", ""
	}
	ok, _ := answer.Get("ok")
	return ok, text(answer, "text")
}

func TestCheckAgainstIEDGivesPlainReasons(t *testing.T) {
	copied, _ := FromGateway("iec104", []any{gatewayPoint("double", 20001, "Q0"), gatewayPoint("float", 30001, "U"),
		gatewayPoint("command_double", 50001, "Q0 command")}, nil, nil)
	item := func(at int, state string, value any) *pyjson.Object {
		return obj("point", copied[at].Point, "state", state, "value", value)
	}
	closed := Mapped{Found: true, Online: true, Value: int64(2), Scale: 1, Source: "FEED1 XCBR1.Pos"}
	cases := []struct {
		item   *pyjson.Object
		mapped Mapped
		ok     any
		text   string
	}{
		{item(0, StateReceived, int64(2)), closed, true, "Both say Closed."},
		{item(0, StateReceived, int64(1)), closed, false, "IED says Closed, master got Open: check the double point mapping."},
		{item(0, StateMissing, nil), closed, nil, "The master has not received this point yet."},
		{item(0, StateReceived, int64(2)), Mapped{Found: true}, nil, "The IED has no value now (it is not running or not connected)."},
		{item(0, StateReceived, int64(2)), Mapped{}, nil, "The gateway list has no double point at IOA 20001 any more."},
		{item(1, StateReceived, 23.0), Mapped{Found: true, Online: true, Value: 230.0, Scale: 0.1}, true, "Both say 23."},
		{item(1, StateReceived, 230.0), Mapped{Found: true, Online: true, Value: 230.0, Scale: 0.1}, false,
			"IED says 23, master got 230: check the scale and the type of this point."},
		{item(2, StateCommand, nil), closed, nil, "A command point: use Send and watch the IED."},
	}
	for number, test := range cases {
		ok, words := checked(t, test.item, test.mapped)
		if ok != test.ok || words != test.text {
			t.Errorf("case %d: got %v %q want %v %q", number, ok, words, test.ok, test.text)
		}
	}
	if ok, _ := checked(t, obj("point", obj("type", "double", "address", int64(1)), "state", StateReceived), closed); ok != "none" {
		t.Fatal("points not copied from the gateway have no check")
	}
}
