package gateway

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// Own points (lane G) are a Go-only extension: the Python oracles cover IED points only, so
// these tests pin the own point rules directly.

func ownRaw(kind string, address int64, extra ...any) *pyjson.Object {
	point := pyjson.NewObject().Set("kind", kind).Set("source", OwnSource).Set("address", address).
		Set("scale", 1.0).Set("description", "")
	for index := 0; index+1 < len(extra); index += 2 {
		point.Set(extra[index].(string), extra[index+1])
	}
	return point
}

func bayIED() *pyjson.Object {
	return pyjson.NewObject().Set("id", "bay").Set("name", "Bay").
		Set("signals", []any{pyjson.NewObject().Set("ref", "LD/XCBR1.Pos.stVal").Set("kind", "Dbpos")}).
		Set("controls", []any{})
}

func encoded(t *testing.T, value any) string {
	t.Helper()
	text, err := pyjson.Dumps(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func TestOwnPointsValidateWithoutIED(t *testing.T) {
	iedPoint := pyjson.NewObject().Set("kind", "double").Set("ied", "bay").Set("ref", "LD/XCBR1.Pos.stVal").
		Set("address", int64(20001))
	points := []any{iedPoint,
		ownRaw("single", 10001, "description", "Pump running", "value", true),
		ownRaw("float", 30001, "id", "own:7", "value", int64(12)),
		ownRaw("integer", 40001),
		ownRaw("double", 20002, "id", "own:7"), // a repeated id gets the next number
		ownRaw("command_double", 50001, "target", "own:8")}
	clean, problems, err := ValidatePoints("iec104", points, []*pyjson.Object{bayIED()})
	if err != nil || len(problems) != 0 {
		t.Fatalf("problems %v %v", problems, err)
	}
	want := `[{"kind": "double", "ied": "bay", "ref": "LD/XCBR1.Pos.stVal", "address": 20001, "scale": 1.0, "description": "", "id": "bay:LD/XCBR1.Pos.stVal:double"}, ` +
		`{"kind": "single", "source": "manual", "ied": "", "ref": "", "address": 10001, "scale": 1.0, "description": "Pump running", "value": 1, "id": "own:8"}, ` +
		`{"kind": "float", "source": "manual", "ied": "", "ref": "", "address": 30001, "scale": 1.0, "description": "", "value": 12.0, "id": "own:7"}, ` +
		`{"kind": "integer", "source": "manual", "ied": "", "ref": "", "address": 40001, "scale": 1.0, "description": "", "value": 0, "id": "own:9"}, ` +
		`{"kind": "double", "source": "manual", "ied": "", "ref": "", "address": 20002, "scale": 1.0, "description": "", "value": 1, "id": "own:10"}, ` +
		`{"kind": "command_double", "source": "manual", "ied": "", "ref": "", "address": 50001, "scale": 1.0, "description": "", "target": "own:8", "id": "own:11"}]`
	if got := encoded(t, clean); got != want {
		t.Fatalf("clean\n got %s\nwant %s", got, want)
	}
}

func TestOwnPointProblems(t *testing.T) {
	points := []any{ownRaw("single", 10001, "value", int64(2)),
		ownRaw("double", 20001, "description", "Breaker"),
		ownRaw("float", 20001), // same IOA as the breaker
		ownRaw("integer", 40001, "value", 1.5),
		ownRaw("command_single", 50001, "target", "own:9"),
		ownRaw("float", 30001, "value", "abc")}
	clean, problems, err := ValidatePoints("iec104", points, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Point 1 (own point): the value must be OFF (0) or ON (1)",
		"Point 3 (own point): address 20001 is already used by point 2",
		"Point 4 (own point): the value must be a whole number from -2147483648 to 2147483647",
		"Point 6 (own point): the value must be a number",
		"Point 5 (own point): the status point it changes (own:9) is not an own single or double point of this list"}
	if strings.Join(problems, "\n") != strings.Join(want, "\n") {
		t.Fatalf("problems\n%s", strings.Join(problems, "\n"))
	}
	if len(clean) != 2 || get(clean[1].(*pyjson.Object), "target") != nil {
		t.Fatalf("clean %s", encoded(t, clean))
	}
}

func TestOwnValueRules(t *testing.T) {
	cases := []struct {
		kind string
		raw  any
		want any
	}{{"single", false, int64(0)}, {"single", int64(1), int64(1)}, {"double", true, int64(2)},
		{"double", int64(3), int64(3)}, {"float", int64(-4), -4.0}, {"float", 2.5, 2.5},
		{"integer", 7.0, int64(7)}, {"command_double", false, int64(1)}}
	for _, item := range cases {
		got, reason := OwnValue(item.kind, item.raw)
		if reason != "" || got != item.want {
			t.Errorf("%s %v: %v %q", item.kind, item.raw, got, reason)
		}
	}
	for _, bad := range []struct {
		kind string
		raw  any
	}{{"single", 0.5}, {"double", int64(4)}, {"float", true}, {"integer", 3e10}, {"float", nil}} {
		if _, reason := OwnValue(bad.kind, bad.raw); reason == "" {
			t.Errorf("%s %v accepted", bad.kind, bad.raw)
		}
	}
}

func TestOwnPointsCSVRoundTrip(t *testing.T) {
	ieds := []*pyjson.Object{bayIED()}
	points := []any{pyjson.NewObject().Set("kind", "double").Set("ied", "bay").Set("ref", "LD/XCBR1.Pos.stVal").
		Set("address", int64(20001)),
		ownRaw("double", 20002, "description", "Fake breaker, \"spare\"", "value", int64(2)),
		ownRaw("float", 30001, "value", 49.95, "scale", 0.5),
		ownRaw("command_double", 50001, "target", "own:1")}
	clean, problems, err := ValidatePoints("iec104", points, ieds)
	if err != nil || len(problems) != 0 {
		t.Fatal(problems, err)
	}
	text, err := ExportCSV(clean, ieds)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(text, "type,address,ied,reference,scale,description,source,value,id,target\r\n") ||
		!strings.Contains(text, "double,20001,Bay,LD/XCBR1.Pos.stVal,1,,,,,\r\n") {
		t.Fatalf("export\n%s", text)
	}
	parsed, problems, err := ImportCSV(text, ieds)
	if err != nil || len(problems) != 0 {
		t.Fatal(problems, err)
	}
	again, problems, err := ValidatePoints("iec104", parsed, ieds)
	if err != nil || len(problems) != 0 {
		t.Fatal(problems, err)
	}
	sorted := []any{clean[3], clean[0], clean[1], clean[2]} // the export sorts by type, then address
	if encoded(t, again) != encoded(t, sorted) {
		t.Fatalf("round trip\n got %s\nwant %s", encoded(t, again), encoded(t, sorted))
	}
	plain, _ := ExportCSV(clean[:1], ieds) // a list without own points keeps the six columns
	if !strings.HasPrefix(plain, "type,address,ied,reference,scale,description\r\n") {
		t.Fatalf("plain export\n%s", plain)
	}
	old := "type,address,ied,reference,scale,description\nsingle,5,Bay,LD/XCBR1.Pos.stVal,1,x\n"
	if rows, problems, _ := ImportCSV(old, ieds); len(rows) != 1 || len(problems) != 0 || IsOwn(rows[0].(*pyjson.Object)) {
		t.Fatalf("old CSV %v %v", rows, problems)
	}
	short := "type,address,ied,reference,scale,description,source,value\nsingle,7,,,,Lamp,own,on\n"
	if rows, problems, _ := ImportCSV(short, ieds); len(problems) != 0 ||
		encoded(t, rows) != `[{"kind": "single", "source": "manual", "address": 7, "ied": "", "ref": "", "scale": 1.0, "description": "Lamp", "value": true}]` {
		t.Fatalf("own row %s %v", encoded(t, rows), problems)
	}
}

func TestNewOwnPointAndKeepOwn(t *testing.T) {
	list := []any{pyjson.NewObject().Set("kind", "float").Set("address", int64(1000)),
		ownRaw("float", 1002, "id", "own:4")}
	point, err := NewOwnPoint("modbus", "float", list)
	if err != nil {
		t.Fatal(err)
	}
	if got := encoded(t, point); got != `{"id": "own:5", "kind": "float", "source": "manual", "ied": "", "ref": "", "address": 1004, "scale": 1.0, "description": "", "value": 0.0}` {
		t.Fatalf("new point %s", got)
	}
	if command, _ := NewOwnPoint("iec104", "command_single", nil); get(command, "address") != int64(50001) ||
		get(command, "id") != "own:1" || get(command, "value") != nil {
		t.Fatalf("command %s", encoded(t, command))
	}
	proposal := []any{pyjson.NewObject().Set("kind", "single").Set("address", int64(10001))}
	merged, notes := KeepOwn("iec104", proposal, []any{ownRaw("single", 10001, "description", "Lamp"),
		ownRaw("float", 30001), pyjson.NewObject().Set("kind", "single").Set("address", int64(3))})
	if len(merged) != 3 || get(merged[1].(*pyjson.Object), "address") != int64(10002) ||
		strings.Join(notes, "|") != `Own point "Lamp" moved from address 10001 to 10002 because a proposed point uses 10001.` {
		t.Fatalf("keep %s %v", encoded(t, merged), notes)
	}
}
