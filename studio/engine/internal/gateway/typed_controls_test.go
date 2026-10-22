package gateway

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// withTypedControls wires ExtendedControls the way the app does, for one test: real IEDs list
// only SPC/DPC in "controls"; BSC, APC and INC come from the stored SCL (ADR-0104).
func withTypedControls(t *testing.T, list []any) {
	t.Helper()
	original := ExtendedControls
	ExtendedControls = func(*pyjson.Object) []any { return list }
	t.Cleanup(func() { ExtendedControls = original })
}

func sclTapIED() *pyjson.Object {
	return pyjson.NewObject().Set("id", "tr").Set("name", "Trafo").
		Set("signals", []any{pyjson.NewObject().Set("ref", "LD/ATCC1.TapChg.valWTr.posVal").Set("kind", "INT8")}).
		Set("controls", []any{pyjson.NewObject().Set("ref", "LD/CSWI1.Pos").Set("cdc", "DPC")})
}

func TestTypedLinksUseTheSclControls(t *testing.T) {
	withTypedControls(t, []any{
		pyjson.NewObject().Set("ref", "LD/ATCC1.TapChg").Set("cdc", "BSC"),
		pyjson.NewObject().Set("ref", "LD/ATCC1.BndCtr").Set("cdc", "APC")})
	points := []any{iedPoint(KindStep, "LD/ATCC1.TapChg", 50001), iedPoint(KindSetpoint, "LD/ATCC1.BndCtr", 55001)}
	if _, problems, err := ValidatePoints("iec104", points, []*pyjson.Object{sclTapIED()}); err != nil || len(problems) != 0 {
		t.Fatalf("typed controls from the SCL are linkable: %v %v", problems, err)
	}
	wrong := []any{iedPoint(KindSetpoint, "LD/ATCC1.TapChg", 55001), iedPoint(KindStep, "LD/NONE1.X", 50002)}
	_, problems, err := ValidatePoints("iec104", wrong, []*pyjson.Object{sclTapIED()})
	text := strings.Join(problems, "\n")
	if err != nil || !strings.Contains(text, "is a BSC and cannot take a set point") ||
		!strings.Contains(text, "Trafo has no controllable object LD/NONE1.X") {
		t.Fatalf("problems %v %v", problems, err)
	}
}

func TestTypedLinksWithoutTheSclAreRefused(t *testing.T) {
	withTypedControls(t, nil)
	points := []any{iedPoint(KindStep, "LD/ATCC1.TapChg", 50001)}
	_, problems, err := ValidatePoints("iec104", points, []*pyjson.Object{sclTapIED()})
	if err != nil || len(problems) != 1 || !strings.Contains(problems[0], "no controllable object") {
		t.Fatalf("problems %v %v", problems, err)
	}
}
