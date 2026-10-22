package masterpoints

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func obj(pairs ...any) *pyjson.Object {
	o := pyjson.NewObject()
	for at := 0; at+1 < len(pairs); at += 2 {
		o.Set(pairs[at].(string), pairs[at+1])
	}
	return o
}

func jsonText(t *testing.T, value any) string {
	t.Helper()
	text, err := pyjson.Compact(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

// The saved list keeps one shape: defaults set, ids given, command and Modbus details derived.
func TestValidateKeepsShapeAndRefusesClashes(t *testing.T) {
	list, err := Validate("iec104", []any{
		obj("type", "double", "address", int64(20001), "name", " Feeder 1 breaker "),
		obj("type", "command_double", "address", int64(50001), "name", "Feeder 1 breaker command"),
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"id":"m1","name":"Feeder 1 breaker","type":"double","address":20001,"scale":1.0,"offset":0.0,"unit":""},` +
		`{"id":"m2","name":"Feeder 1 breaker command","type":"command_double","address":50001,"scale":1.0,"offset":0.0,` +
		`"unit":"","command":"double","select":true,"value":2}]`
	if got := jsonText(t, list); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	_, err = Validate("iec104", []any{obj("type", "double", "address", int64(20001)), obj("type", "single", "address", int64(20001))})
	if err == nil || !strings.Contains(err.Error(), "Point 2: IOA 20001 is already used by point 1") {
		t.Fatalf("an IOA is one point in IEC 104: %v", err)
	}
	for _, bad := range []*pyjson.Object{obj("type", "M_DP", "address", int64(1)), obj("type", "double", "address", int64(0)),
		obj("type", "double", "address", 1.5), obj("type", "command_double", "address", int64(9), "value", int64(3)),
		obj("type", "float", "address", int64(9), "scale", int64(0))} {
		if _, err := Validate("iec104", []any{bad}); err == nil {
			t.Errorf("%s must be refused", jsonText(t, bad))
		}
	}
	modbus, err := Validate("modbus", []any{obj("type", "holding_float32", "address", int64(100), "word_order", "little"),
		obj("type", "input_float32", "address", int64(100)), obj("type", "coil", "address", int64(3), "select", true)})
	if err != nil {
		t.Fatal(err)
	}
	if got := jsonText(t, modbus[0]); !strings.Contains(got, `"table":"holding_registers","data_type":"float32","registers":2,`+
		`"word_order":"little","command":"registers_float","select":false,"value":0`) {
		t.Fatalf("Modbus details: %s", got)
	}
	if got := jsonText(t, modbus[2]); !strings.Contains(got, `"select":false`) {
		t.Fatalf("Modbus has no select before operate: %s", got)
	}
	if _, err := Validate("modbus", []any{obj("type", "holding_float32", "address", int64(65535))}); err == nil {
		t.Fatal("a float32 needs two registers: 65535 is the last address")
	}
	if _, err := Validate("modbus", []any{obj("type", "holding_int16", "address", int64(7)),
		obj("type", "holding_float32", "address", int64(6))}); err == nil {
		t.Fatal("a float32 at 6 uses register 7 too")
	}
	dnp3, err := Validate("dnp3", []any{obj("type", "binary", "address", int64(0)), obj("type", "analog", "address", int64(0)),
		obj("type", "crob", "address", int64(0)), obj("type", "crob_trip_close", "address", int64(1), "id", "m7"),
		obj("type", "double_binary", "address", int64(0))})
	if err != nil || len(dnp3) != 5 || text(dnp3[3].(*pyjson.Object), "code") != "trip_close" ||
		text(dnp3[3].(*pyjson.Object), "id") != "m7" || text(dnp3[0].(*pyjson.Object), "id") != "m8" {
		t.Fatalf("DNP3 numbers each group by itself: %v %s", err, jsonText(t, dnp3))
	}
}

func TestRangeStepsByRegistersAndStopsOnUsedAddresses(t *testing.T) {
	results := Range("modbus", "holding_float32", 100, 3, "Feeder current", nil)
	if !OK(results) || Address(results[2].Point) != 104 || text(results[1].Point, "name") != "Feeder current 2" ||
		text(results[2].Point, "id") != "m3" {
		t.Fatalf("range: %s", jsonText(t, PlanJSON(results)))
	}
	current, _ := Validate("iec104", []any{obj("type", "single", "address", int64(10002), "name", "Alarm")})
	clash := Range("iec104", "single", 10001, 3, "", current)
	if OK(clash) || clash[1].Problem != `Row 2: IOA 10002 is already used by "Alarm"` || clash[0].Problem != "" {
		t.Fatalf("a used address stops the plan with a reason: %s", jsonText(t, PlanJSON(clash)))
	}
	if plan := PlanJSON(clash); len(plan.Keys()) != 3 || len(field(plan, "points").([]any)) != 0 {
		t.Fatal("a plan that is not ok adds nothing")
	}
	if one := One("iec104", obj("type", "double", "address", int64(20001), "name", "Q0"), current); !OK(one) ||
		text(one[0].Point, "id") != "m2" {
		t.Fatalf("one point: %s", jsonText(t, PlanJSON(one)))
	}
}

func TestFromReceivedAddsWhatTheDeviceSent(t *testing.T) {
	rows := []any{obj("key", "double:20005", "group", "double", "address", int64(20005), "type", "M_DP_TB_1", "label", "Q5"),
		obj("key", "holding_register:7", "group", "holding_register", "address", int64(7), "type", "uint32")}
	plan := FromReceived("iec104", rows, []string{"double:20005"}, nil)
	if !OK(plan) || text(plan[0].Point, "type") != "double" || text(plan[0].Point, "name") != "Q5" {
		t.Fatalf("from received: %s", jsonText(t, PlanJSON(plan)))
	}
	unknown := FromReceived("modbus", rows, []string{"holding_register:7"}, nil)
	if OK(unknown) || !strings.Contains(unknown[0].Problem, "no list type") {
		t.Fatalf("a uint32 register has no list type: %s", jsonText(t, PlanJSON(unknown)))
	}
}
