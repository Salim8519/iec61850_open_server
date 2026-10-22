package addpoints

import (
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// TestSmartProposesTypedCommands (0178, C1): smart Propose adds a tap changer (BSC) as a step
// command with its tap position, and an APC as a set point with its measured value, each
// paired by address; the plain Propose (Python parity) still proposes SPC/DPC only.
func TestSmartProposesTypedCommands(t *testing.T) {
	original := gateway.ExtendedControls
	gateway.ExtendedControls = func(ied *pyjson.Object) []any {
		if text(ied, "id") != "t1" {
			return nil
		}
		return []any{control("TR1LD0/ATCC1.TapChg", "BSC", "TR1LD0/ATCC1.TapChg.valWTr.posVal"),
			control("TR1LD0/ATCC1.BndCtr", "APC", "TR1LD0/ATCC1.BndCtr.mxVal.f"),
			control("TR1LD0/LLN0.Mod", "ENC", "TR1LD0/LLN0.Mod.stVal")}
	}
	t.Cleanup(func() { gateway.ExtendedControls = original })
	transformer := pyjson.NewObject().Set("id", "t1").Set("name", "TR1").Set("role", "simulated").
		Set("signals", []any{signal("TR1LD0/ATCC1.TapChg.valWTr.posVal", "ST", "INT8", "Tap position"),
			signal("TR1LD0/ATCC1.BndCtr.mxVal.f", "MX", "FLOAT32", "Band centre")}).
		Set("controls", []any{}).Set("datasets", []any{})
	ieds := []*pyjson.Object{feeder(), transformer}
	plain, _, err := gateway.Propose(ieds, "iec104")
	if err != nil {
		t.Fatal(err)
	}
	if _, found := byID(plain)["t1:TR1LD0/ATCC1.TapChg:command_step"]; found {
		t.Fatal("the plain Propose must stay the Python selection")
	}
	for _, protocol := range []string{"iec104", "dnp3", "modbus"} {
		points, _, summary, err := Smart(ieds, protocol, SmartOptions{})
		if err != nil {
			t.Fatal(err)
		}
		found := byID(points)
		step, tap := found["t1:TR1LD0/ATCC1.TapChg:command_step"], found["t1:TR1LD0/ATCC1.TapChg.valWTr.posVal:step_position"]
		setpoint := found["t1:TR1LD0/ATCC1.BndCtr:command_setpoint"]
		if protocol == "modbus" && step == nil && tap == nil && setpoint != nil { // Modbus has no step command
			continue
		}
		if step == nil || tap == nil || setpoint == nil || found["t1:TR1LD0/LLN0.Mod:command_single"] != nil {
			t.Fatalf("%s points %s", protocol, pointJSON(t, points))
		}
		if text(step, "description") != "Tap position" {
			t.Fatalf("%s name %s", protocol, pointJSON(t, step))
		}
		for _, point := range []*pyjson.Object{step, tap, setpoint} { // steps share the command block
			if at, _ := point.Get("address"); at.(int64) < 0 {
				t.Fatalf("%s address %s", protocol, pointJSON(t, point))
			}
		}
		if _, problems, err := gateway.ValidatePoints(protocol, points, ieds); err != nil || len(problems) != 0 {
			t.Fatalf("%s validate %v %v", protocol, problems, err)
		}
		if summary[1].Offered != 2 { // the two typed commands (statuses are added later)
			t.Fatalf("%s summary %+v", protocol, summary)
		}
	}
}
