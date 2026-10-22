package runtime

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// Typed point panel commands (ADR-0108) on the fake workers: an SBO single command, an SBO
// integer command and a direct list command of BayA (typed list as ADR-0104's SCL reader gives it).
// withExtendedControls gives every IED this typed list instead of the one read from its SCL.
func withExtendedControls(t *testing.T, list []any) {
	t.Helper()
	original := extendedControlsOf
	extendedControlsOf = func(*pyjson.Object) []any { return list }
	t.Cleanup(func() { extendedControlsOf = original })
}

func typedPanelRun(t *testing.T, portBase int64) *Coordinator {
	t.Helper()
	project, a, _ := fakeProject(t, 13, portBase, false)
	ied := projectIeds(project)[0]
	for _, item := range pyval.List(ied, "controls") {
		if control := item.(*pyjson.Object); pyval.Str(control, "ref") == "BayACTRL/GGIO1.SPCSO1" {
			control.Set("model", int64(2))
		}
	}
	withExtendedControls(t, []any{
		pyval.Obj("ref", "BayACTRL/GGIO1.ISCSO1", "status", "BayACTRL/GGIO1.ISCSO1.stVal", "cdc", "INC", "value_kind", "int",
			"ctl_val_kind", "INT32", "model", int64(2), "sbo_timeout_ms", int64(30000), "min", int64(0), "max", int64(10),
			"step", int64(1), "labels", pyval.Obj()),
		pyval.Obj("ref", "BayACTRL/LLN0.Mod", "status", "BayACTRL/LLN0.Mod.stVal", "cdc", "ENC", "value_kind", "enum", "ctl_val_kind", "Enum",
			"model", int64(1), "sbo_timeout_ms", int64(0), "min", nil, "max", nil, "step", nil,
			"labels", pyval.Obj("on", int64(1), "blocked", int64(2), "test", int64(3))),
	})
	rows := []any{panelRow("spc", "command", a, "BayACTRL/GGIO1.SPCSO1", nil), panelRow("inc", "command", a, "BayACTRL/GGIO1.ISCSO1", nil),
		panelRow("enc", "command", a, "BayACTRL/LLN0.Mod", nil), panelRow("s", "status", a, "BayACTRL/XCBR1.Loc.stVal", true)}
	nodes := pyval.List(pyval.Map(project, "diagram"), "nodes")
	pyval.Map(project, "diagram").Set("nodes", append(nodes, panelNode(a, rows...)))
	c := New(Options{NativeDir: fakeNativeDir(t), DataDir: testutil.TempDir(t), Bus: &MemoryBus{}, Project: NewMemoryProject(project)})
	t.Cleanup(c.Close)
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPointPanelSelectOperate(t *testing.T) {
	c := typedPanelRun(t, 23400)
	answer, err := c.EquipmentTest("pp", "select", true, "spc")
	if err != nil || pyval.Get(answer, "selected") != true || pyval.Get(answer, "expires_ms") != int64(7000) ||
		pyval.Get(answer, "model") != "sbo-normal" {
		t.Fatalf("select: %v %v", answer, err)
	}
	if _, err = c.EquipmentTest("pp", "send", false, "spc"); err == nil || !strings.Contains(err.Error(), "is selected for true") {
		t.Fatalf("another value than the selected one is refused, got %v", err)
	}
	if answer, err = c.EquipmentTest("pp", "send", true, "spc"); err != nil || pyval.Get(answer, "value") != true {
		t.Fatalf("operate within the window: %v %v", answer, err)
	}
	if answer, _ = c.EquipmentTest("pp", "cancel", nil, "spc"); pyval.Get(answer, "cancelled") != false {
		t.Fatalf("the operate used the selection up: %v", answer)
	}
	if _, err = c.EquipmentTest("pp", "select", nil, "s"); err == nil || !strings.Contains(err.Error(), "cannot be selected") {
		t.Fatalf("a status point is not selected, got %v", err)
	}
}

func TestPointPanelTypedValues(t *testing.T) {
	c := typedPanelRun(t, 23410)
	if _, err := c.EquipmentTest("pp", "select", int64(11), "inc"); err == nil || !strings.Contains(err.Error(), "up to 10") {
		t.Fatalf("an INC value above max is refused before anything is selected, got %v", err)
	}
	if answer, err := c.EquipmentTest("pp", "select", int64(4), "inc"); err != nil || pyval.Get(answer, "selected") != true {
		t.Fatalf("an SBO INC value is selected through the typed path: %v %v", answer, err)
	}
	if _, err := c.EquipmentTest("pp", "send", int64(4), "inc"); err != nil {
		t.Fatalf("the selected INC value is operated: %v", err)
	}
	if _, err := c.EquipmentTest("pp", "send", "off", "enc"); err == nil || !strings.Contains(err.Error(), "on, blocked, test") {
		t.Fatalf("an unknown label is refused, got %v", err)
	}
	if _, err := c.EquipmentTest("pp", "select", "on", "enc"); err == nil || !strings.Contains(err.Error(), "needs no Select") {
		t.Fatalf("a direct ENC is sent without Select, got %v", err)
	}
	if _, err := c.EquipmentTest("pp", "send", "test", "enc"); err != nil {
		t.Fatalf("a valid label is operated through the typed path: %v", err)
	}
}
