package addpoints

import (
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// Typed controls (ADR-0104) come from the stored SCL through gateway.ExtendedControls; Add points
// offers BSC/ISC for a step, APC/INC for a set point, and pairs each with its status (ADR-0107).
func TestTypedCandidatesAndStatus(t *testing.T) {
	original := gateway.ExtendedControls
	gateway.ExtendedControls = func(*pyjson.Object) []any {
		return []any{control("TRLD0/ATCC1.TapChg", "BSC", "TRLD0/ATCC1.TapChg.valWTr.posVal"),
			control("TRLD0/ATCC1.BndCtr", "APC", "TRLD0/ATCC1.BndCtr.mxVal.f"),
			control("TRLD0/LLN0.Mod", "ENC", "TRLD0/LLN0.Mod.stVal")}
	}
	t.Cleanup(func() { gateway.ExtendedControls = original })
	ied := NewIED(pyjson.NewObject().Set("id", "t1").Set("name", "TR1").
		Set("signals", []any{signal("TRLD0/ATCC1.TapChg.valWTr.posVal", "ST", "INT8", "Tap"),
			signal("TRLD0/ATCC1.BndCtr.mxVal.f", "MX", "FLOAT32", "Band")}).
		Set("controls", []any{control("TRLD0/CSWI1.Pos", "DPC", "")}))
	steps, _ := Candidates(ied, gateway.KindStep, "", nil)
	if len(steps) != 1 || steps[0].Ref != "TRLD0/ATCC1.TapChg" || steps[0].StatusRef != "TRLD0/ATCC1.TapChg.valWTr.posVal" {
		t.Fatalf("step candidates %+v", steps)
	}
	setpoints, _ := Candidates(ied, gateway.KindSetpoint, "", nil)
	if len(setpoints) != 1 || setpoints[0].CDC != "APC" {
		t.Fatalf("set point candidates %+v", setpoints)
	}
	if singles, _ := Candidates(ied, "command_single", "", nil); len(singles) != 0 {
		t.Fatalf("an ENC is not offered as a single command: %+v", singles)
	}
	if _, kind := ied.StatusOf(ied.Control("TRLD0/ATCC1.TapChg")); kind != gateway.KindStepPosition {
		t.Fatalf("a BSC status is a step position, got %q", kind)
	}
	if _, kind := ied.StatusOf(ied.Control("TRLD0/ATCC1.BndCtr")); kind != "float" {
		t.Fatalf("an APC status is a float, got %q", kind)
	}
	if problem := fitProblem(ied, "command_single", "TRLD0/LLN0.Mod"); problem == "" {
		t.Fatal("an ENC as a single command is refused in words")
	}
}
