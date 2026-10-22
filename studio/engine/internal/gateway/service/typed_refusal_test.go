package service

import (
	"strings"
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// A SCADA step or set point to a typed control (ADR-0104, read from the stored SCL) is not refused
// as "no controllable object" (found in the 0173 run: every TI 47 / TI 50 got a negative ACT_CON).
func TestTypedCommandControlFound(t *testing.T) {
	original := gateway.ExtendedControls
	gateway.ExtendedControls = func(*pyjson.Object) []any {
		return []any{pyjson.NewObject().Set("ref", "TR/ATCC1.TapChg").Set("cdc", "BSC")}
	}
	t.Cleanup(func() { gateway.ExtendedControls = original })
	ied := pyjson.NewObject().Set("id", "t1").Set("name", "TR1").Set("role", "simulated").Set("state", "running").
		Set("controls", []any{})
	step := pyjson.NewObject().Set("kind", gateway.KindStep).Set("ref", "TR/ATCC1.TapChg")
	if reason := Refusal(step, ied, true); reason != "" {
		t.Fatalf("a typed control is found: %q", reason)
	}
	if cdc := text(findControl(ied, "TR/ATCC1.TapChg"), "cdc"); cdc != "BSC" {
		t.Fatalf("its class is known for the value check, got %q", cdc)
	}
	missing := pyjson.NewObject().Set("kind", gateway.KindStep).Set("ref", "TR/NONE.X")
	if reason := Refusal(missing, ied, true); !strings.Contains(reason, "no controllable object") {
		t.Fatalf("an unknown control is still refused: %q", reason)
	}
}
