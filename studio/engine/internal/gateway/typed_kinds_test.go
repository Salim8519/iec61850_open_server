package gateway

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// Typed kinds (ADR-0107) are Go-only: the Python oracles never produce them, so these tests
// pin their rules directly. The six Python kinds keep their oracle tests.

func tapIED() *pyjson.Object {
	control := func(ref, cdc string) *pyjson.Object {
		return pyjson.NewObject().Set("ref", ref).Set("cdc", cdc).Set("status", ref+".stVal")
	}
	return pyjson.NewObject().Set("id", "tr").Set("name", "Trafo").
		Set("signals", []any{pyjson.NewObject().Set("ref", "LD/ATCC1.TapChg.valWTr.posVal").Set("kind", "INT8"),
			pyjson.NewObject().Set("ref", "LD/MMXU1.TotW.mag.f").Set("kind", "FLOAT32")}).
		Set("controls", []any{control("LD/ATCC1.TapChg", "BSC"), control("LD/ATCC1.BndCtr", "APC"),
			control("LD/ATCC1.ParOp", "INC"), control("LD/CSWI1.Pos", "DPC")})
}

func iedPoint(kind, ref string, address int64, extra ...any) *pyjson.Object {
	point := pyjson.NewObject().Set("kind", kind).Set("ied", "tr").Set("ref", ref).Set("address", address)
	for index := 0; index+1 < len(extra); index += 2 {
		point.Set(extra[index].(string), extra[index+1])
	}
	return point
}

func TestTypedKindsValidate(t *testing.T) {
	points := []any{
		iedPoint(KindStepPosition, "LD/ATCC1.TapChg.valWTr.posVal", 45001),
		iedPoint(KindStep, "LD/ATCC1.TapChg", 50001),
		iedPoint(KindSetpoint, "LD/ATCC1.BndCtr", 55001),
		iedPoint(KindSetpoint, "LD/ATCC1.ParOp", 55002, "format", "scaled")}
	clean, problems, err := ValidatePoints("iec104", points, []*pyjson.Object{tapIED()})
	if err != nil || len(problems) != 0 {
		t.Fatalf("problems %v %v", problems, err)
	}
	if got := get(clean[2].(*pyjson.Object), "format"); got != "float" {
		t.Fatalf("default 104 set point format %v", got)
	}
	if got := get(clean[3].(*pyjson.Object), "id"); got != "tr:LD/ATCC1.ParOp:command_setpoint" {
		t.Fatalf("id %v", got)
	}
}

func TestTypedKindProblems(t *testing.T) {
	points := []any{
		iedPoint(KindStepPosition, "LD/MMXU1.TotW.mag.f", 1),
		iedPoint(KindStep, "LD/CSWI1.Pos", 2),
		iedPoint(KindSetpoint, "LD/ATCC1.TapChg", 3),
		iedPoint(KindSetpoint, "LD/ATCC1.BndCtr", 4, "format", "int16"),
		iedPoint(KindStep, "LD/ATCC1.TapChg", 5)}
	_, problems, err := ValidatePoints("iec104", points, []*pyjson.Object{tapIED()})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"Point 1 (Trafo LD/MMXU1.TotW.mag.f): LD/MMXU1.TotW.mag.f is not a whole number and cannot be sent as a step position (use valWTr.posVal)",
		"Point 2 (Trafo LD/CSWI1.Pos): LD/CSWI1.Pos is a DPC and cannot take a regulating step (lower / raise) (it needs BSC or ISC)",
		"Point 3 (Trafo LD/ATCC1.TapChg): LD/ATCC1.TapChg is a BSC and cannot take a set point (it needs APC or INC)",
		"Point 4 (Trafo LD/ATCC1.BndCtr): the set point format must be one of float, scaled, normalized"}
	if strings.Join(problems, "\n") != strings.Join(want, "\n") {
		t.Fatalf("problems\n%s", strings.Join(problems, "\n"))
	}
	_, problems, _ = ValidatePoints("modbus", []any{iedPoint(KindStep, "LD/ATCC1.TapChg", 1)}, []*pyjson.Object{tapIED()})
	if len(problems) != 1 || !strings.Contains(problems[0], "Modbus has no regulating step command") {
		t.Fatalf("modbus step %v", problems)
	}
}

func TestModbusSetPointWidthFollowsFormat(t *testing.T) {
	points := []any{
		iedPoint(KindSetpoint, "LD/ATCC1.BndCtr", 9000, "format", "float", "word_order", "low_first"),
		iedPoint(KindSetpoint, "LD/ATCC1.ParOp", 9001, "format", "int16"), // inside the float's two registers
		iedPoint(KindSetpoint, "LD/ATCC1.ParOp", 9002, "format", "uint16", "word_order", "middle")}
	clean, problems, err := ValidatePoints("modbus", points, []*pyjson.Object{tapIED()})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Point 2 (Trafo LD/ATCC1.ParOp): address 9001 is already used by point 1",
		"Point 3 (Trafo LD/ATCC1.ParOp): the word order must be high_first or low_first"}
	if strings.Join(problems, "\n") != strings.Join(want, "\n") || len(clean) != 1 {
		t.Fatalf("problems\n%s", strings.Join(problems, "\n"))
	}
	if get(clean[0].(*pyjson.Object), "word_order") != "low_first" {
		t.Fatalf("word order lost: %v", encoded(t, clean[0]))
	}
	// A DNP3 set point has its own index space: index 0 is free next to a CROB at 0.
	_, problems, _ = ValidatePoints("dnp3", []any{iedPoint(KindStep, "LD/ATCC1.TapChg", 0),
		iedPoint(KindSetpoint, "LD/ATCC1.BndCtr", 0)}, []*pyjson.Object{tapIED()})
	if len(problems) != 0 {
		t.Fatalf("dnp3 spaces %v", problems)
	}
}

func TestTypedCSVRoundTrip(t *testing.T) {
	ieds := []*pyjson.Object{tapIED()}
	clean, _, _ := ValidatePoints("modbus", []any{
		iedPoint(KindSetpoint, "LD/ATCC1.BndCtr", 9000, "format", "int32", "word_order", "low_first"),
		iedPoint(KindStepPosition, "LD/ATCC1.TapChg.valWTr.posVal", 8000)}, ieds)
	text, err := ExportCSV(clean, ieds)
	if err != nil {
		t.Fatal(err)
	}
	want := "type,address,ied,reference,scale,description,format,word_order\r\n" +
		"command_setpoint,9000,Trafo,LD/ATCC1.BndCtr,1,,int32,low_first\r\n" +
		"step_position,8000,Trafo,LD/ATCC1.TapChg.valWTr.posVal,1,,,\r\n"
	if text != want {
		t.Fatalf("csv\n%q\nwant\n%q", text, want)
	}
	points, problems, err := ImportCSV(text, ieds)
	if err != nil || len(problems) != 0 {
		t.Fatalf("import %v %v", problems, err)
	}
	again, _, _ := ValidatePoints("modbus", points, ieds)
	if encoded(t, again) != encoded(t, clean) {
		t.Fatalf("round trip\n%s\n%s", encoded(t, again), encoded(t, clean))
	}
	// A list without set points exports exactly as before (no format columns).
	plain, _ := ExportCSV(clean[1:], ieds)
	if !strings.HasPrefix(plain, "type,address,ied,reference,scale,description\r\n") {
		t.Fatalf("plain header %q", plain)
	}
}

func TestOwnTypedPoints(t *testing.T) {
	points := []any{ownRaw(KindStepPosition, 45001, "value", int64(3), "id", "own:1"),
		ownRaw(KindStep, 50001, "target", "own:1", "id", "own:2"),
		ownRaw(KindSetpoint, 55001, "target", "own:2"), // a command cannot be driven
		ownRaw(KindStepPosition, 45002, "value", int64(64))}
	clean, problems, err := ValidatePoints("iec104", points, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Point 4 (own point): the tap position must be a whole number from -64 to 63",
		"Point 3 (own point): the point it sets (own:2) is not an own measured value, whole number or tap position of this list"}
	if strings.Join(problems, "\n") != strings.Join(want, "\n") || len(clean) != 3 {
		t.Fatalf("problems\n%s", strings.Join(problems, "\n"))
	}
	if get(clean[1].(*pyjson.Object), "target") != "own:1" {
		t.Fatalf("step target lost")
	}
	if !KnownKind(KindStep) || KnownKind("analog") || contains(Kinds, KindStep) {
		t.Fatal("kind lists")
	}
}
