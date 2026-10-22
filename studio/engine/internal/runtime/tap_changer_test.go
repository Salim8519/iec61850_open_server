package runtime

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// Tap-changer transformer (ADR-0109) on the fake workers: BayA simulated, BayB real. The fake
// BayA has one SPC, so raise and lower share it here (the project package refuses that; the
// runtime only routes). Its LLN0.Health.stVal (INT32 ST) stands for the tap position.

const (
	fakeTap   = "BayACTRL/LLN0.Health.stVal"
	fakeSPC   = "BayACTRL/GGIO1.SPCSO1"
	fakeVolts = "BayACTRL/GGIO1.AnIn1.mag.f"
)

func tapData(ied, mode string, avr *pyjson.Object) *pyjson.Object {
	tap := pyval.Obj("tap_source", pyval.Obj("ied_id", ied, "ref", fakeTap, "report_ref", nil), "command_mode", mode,
		"step_control", nil, "raise_control", nil, "lower_control", nil, "mode_source", nil, "mode_control", nil,
		"auto_value", int64(1), "manual_value", int64(0),
		"voltage_source", pyval.Obj("ied_id", ied, "ref", fakeVolts, "report_ref", nil),
		"min_tap", int64(1), "max_tap", int64(17), "nominal_tap", int64(9), "avr", avr)
	if mode == "step" {
		tap.Set("step_control", pyval.Obj("ied_id", ied, "ref", "BayACTRL/YLTC1.TapChg"))
	} else {
		tap.Set("raise_control", pyval.Obj("ied_id", ied, "ref", fakeSPC))
		tap.Set("lower_control", pyval.Obj("ied_id", ied, "ref", fakeSPC))
	}
	return pyval.Obj("kind", tapTransformerKind, "label", "T1", "rotation", int64(0), "state_mapping", nil,
		"control_binding", nil, "relay_statuses", []any{}, "tap_changer", tap)
}

func avrOn(sourceKV, delay float64) *pyjson.Object {
	return pyval.Obj("enabled", true, "target_kv", 11.0, "bandwidth_pct", 1.5, "delay_s", delay, "step_pct", 1.25,
		"hv_rated_kv", 132.0, "lv_rated_kv", 11.0, "source_kv", sourceKV)
}

// startTap starts BayA (and BayB when real) with one transformer whose points are on BayA, or on
// BayB when real is set, and sets the tap to start.
func startTap(t *testing.T, portBase int64, mode string, avr *pyjson.Object, real bool, start int64) (*Coordinator, *MemoryBus, string) {
	t.Helper()
	project, a, b := fakeProject(t, 11, portBase, false)
	target := a
	if real {
		projectIeds(project)[1].Set("role", "real")
		target = b
	}
	if mode == "step" { // a typed BSC control as change 0167 (ADR-0104) lists it
		withExtendedControls(t, []any{pyval.Obj("ref", "BayACTRL/YLTC1.TapChg", "cdc", "BSC", "value_kind", "step",
			"model", int64(1), "status", "BayACTRL/YLTC1.TapChg.valWTr", "min", int64(1), "max", int64(17))})
	}
	data := tapData(target, mode, avr)
	if real { // BayB has the same model under its own name
		tap := pyval.Map(data, "tap_changer")
		for _, key := range []string{"tap_source", "voltage_source", "raise_control", "lower_control"} {
			binding := pyval.Map(tap, key)
			binding.Set("ref", strings.Replace(pyval.Str(binding, "ref"), "BayA", "BayB", 1))
		}
	}
	diagram := pyval.Map(project, "diagram")
	node := pyval.Obj("id", "t1", "type", "equipment", "position", pyval.Obj("x", int64(0), "y", int64(0)), "data", data)
	diagram.Set("nodes", append(pyval.List(diagram, "nodes"), node))
	bus := &MemoryBus{}
	c := New(Options{NativeDir: fakeNativeDir(t), DataDir: testutil.TempDir(t), Bus: bus, Project: NewMemoryProject(project)})
	t.Cleanup(c.Close)
	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	if !real {
		if _, err := c.Update(a, fakeTap, start, int64(0), "manual"); err != nil {
			t.Fatal(err)
		}
	}
	return c, bus, a
}

func tapEvents(bus *MemoryBus, prefix string) []*pyjson.Object {
	var found []*pyjson.Object
	for _, event := range bus.Events() {
		if strings.HasPrefix(pyval.Str(event, "kind"), prefix) {
			found = append(found, event)
		}
	}
	return found
}

func TestTapRaiseWithSPCPairMovesTheSimulatedTap(t *testing.T) {
	c, bus, a := startTap(t, 23600, "spc_pair", avrOn(132, 30), false, 9)
	answer, err := c.Operate("t1", nil, "raise")
	if err != nil || pyval.Get(answer, "tap") != int64(10) || pyval.Get(answer, "feedback") != true {
		t.Fatalf("raise: %v %v", answer, err)
	}
	if value, _ := c.Value(a, fakeTap); value != int64(10) {
		t.Fatalf("the simulated tap is %v", value)
	}
	if _, err = c.Operate("t1", nil, "lower"); err != nil {
		t.Fatal(err)
	}
	if value, _ := c.Value(a, fakeTap); value != int64(9) {
		t.Fatalf("after lower the tap is %v", value)
	}
	events := tapEvents(bus, "tap_command_")
	if len(events) != 4 || pyval.Str(events[1], "kind") != "tap_command_applied" ||
		pyval.Str(events[0], "correlation_id") != pyval.Str(events[1], "correlation_id") ||
		pyval.Str(events[0], "origin") != "studio_api" || pyval.Str(events[0], "ied_id") != a {
		t.Fatalf("events %v", events)
	}
}

func TestTapRaiseRefusals(t *testing.T) {
	c, _, _ := startTap(t, 23700, "spc_pair", avrOn(132, 30), false, 17)
	if _, err := c.Operate("t1", nil, "raise"); err == nil || !strings.Contains(err.Error(), "highest tap (17)") {
		t.Fatalf("raise at max: %v", err)
	}
	if _, err := c.Operate("t1", nil, "spin"); err == nil || !strings.Contains(err.Error(), "Unknown transformer action") {
		t.Fatalf("unknown action: %v", err)
	}
	if _, err := c.Operate("t1", nil, "auto"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Operate("t1", nil, "lower"); err == nil || !strings.Contains(err.Error(), "is in Auto") {
		t.Fatalf("a manual step in Auto must be refused: %v", err)
	}
}

// The step binding goes through the typed path (ADR-0104): a raise reaches the control program as
// one "higher" step and is logged as requested, then applied.
func TestTapStepCommandUsesTypedControls(t *testing.T) {
	c, bus, _ := startTap(t, 23800, "step", avrOn(132, 30), false, 9)
	if _, err := c.Operate("t1", nil, "raise"); err != nil {
		t.Fatalf("BSC step through the typed path: %v", err)
	}
	if applied := tapEvents(bus, "tap_command_applied"); len(applied) != 1 {
		t.Fatalf("the step is logged as applied: %v", applied)
	}
}

func TestAvrLowersAHighVoltageOnItsOwn(t *testing.T) {
	c, bus, a := startTap(t, 23900, "spc_pair", avrOn(140, 0), false, 9) // 140 kV: LV 11.67 kV, +6 %
	if _, err := c.Operate("t1", nil, "auto"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if value, _ := c.Value(a, fakeTap); value == int64(5) { // 140 kV x ratio(5) = 11.11 kV, in band
			break
		}
		if time.Now().After(deadline) {
			value, _ := c.Value(a, fakeTap)
			t.Fatalf("the regulator did not settle: tap %v, events %v", value, tapEvents(bus, "avr_"))
		}
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond)
	if value, _ := c.Value(a, fakeTap); value != int64(5) {
		t.Fatalf("the regulator must stop in band, tap %v", value)
	}
	if value, _ := c.Value(a, fakeVolts); value == nil || value.(float64) < 11.0 || value.(float64) > 11.2 {
		t.Fatalf("the LV voltage point holds %v", value)
	}
	steps := 0
	for _, event := range tapEvents(bus, "avr_decision") {
		if pyval.Str(event, "event") == "step" {
			steps++
			if pyval.Str(event, "action") != "lower" || pyval.Str(event, "correlation_id") == "" || pyval.Str(event, "ied_id") != a {
				t.Fatalf("step %v", event)
			}
		}
	}
	if steps != 4 {
		t.Fatalf("4 lower steps from 9 to 5, got %d", steps)
	}
	for _, event := range tapEvents(bus, "tap_command_requested") {
		if pyval.Str(event, "origin") != "avr" {
			t.Fatalf("regulator commands carry origin avr: %v", event)
		}
	}
}

func TestAvrNeverCommandsARealDevice(t *testing.T) {
	c, bus, _ := startTap(t, 24000, "spc_pair", avrOn(140, 0), true, 9)
	if _, err := c.Operate("t1", nil, "auto"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(400 * time.Millisecond)
	if sent := tapEvents(bus, "tap_command_"); len(sent) != 0 {
		t.Fatalf("no command may reach a real IED: %v", sent)
	}
	blocked := false
	for _, event := range tapEvents(bus, "avr_decision") {
		blocked = blocked || (pyval.Str(event, "event") == "blocked" && pyval.Str(event, "reason") == "real_ied")
	}
	if !blocked {
		t.Fatal("the blocked regulator says why in the log")
	}
	if _, err := c.Operate("t1", nil, "manual"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Operate("t1", nil, "raise"); err == nil || !strings.Contains(err.Error(), "real device") {
		t.Fatalf("a manual raise on a real IED: %v", err)
	}
}

func TestPlantStatesIncludeTapRows(t *testing.T) {
	p := newPlantStates(&Coordinator{})
	node := pyval.Obj("id", "t1", "data", tapData("ied1", "spc_pair", avrOn(132, 30)))
	p.reset(pyval.Obj("nodes", []any{node}, "edges", []any{}))
	if len(p.order) != 2 || p.order[0] != "t1:tap" || p.order[1] != "t1:lv" {
		t.Fatalf("rows %v: tap and LV voltage (no mode status is bound)", p.order)
	}
}
