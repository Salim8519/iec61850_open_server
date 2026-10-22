package runtime

import (
	"reflect"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// Point panel (ADR-0102): live rows, the three request actions and the events, on the fake
// workers of the runtime tests (BayA simulated, BayB real).

func panelRow(id, kind, ied, ref string, healthy any) any {
	row := pyval.Obj("id", id, "label", "Point "+id, "group", "", "type", kind, "source", nil, "control", nil,
		"unit", "", "scale", 1.0, "offset", 0.0, "decimals", int64(2), "pinned", false, "healthy", healthy)
	if kind == "command" {
		row.Set("control", pyval.Obj("ied_id", ied, "ref", ref))
	} else {
		row.Set("source", pyval.Obj("ied_id", ied, "ref", ref, "report_ref", nil))
	}
	return row
}

func panelNode(ied string, rows ...any) *pyjson.Object {
	return pyval.Obj("id", "pp", "type", "equipment", "position", pyval.Obj("x", int64(0), "y", int64(0)),
		"data", pyval.Obj("kind", pointPanelKind, "label", "Bay panel", "rotation", int64(0), "state_mapping", nil,
			"control_binding", nil, "relay_statuses", []any{}, "ied_id", ied, "points", rows))
}

func panelRows(ied string) []any {
	return []any{
		panelRow("s", "status", ied, "BayACTRL/XCBR1.Loc.stVal", true),
		panelRow("d", "double", ied, "BayACTRL/XCBR1.Pos.stVal", int64(2)),
		panelRow("v", "value", ied, "BayACTRL/GGIO1.AnIn1.mag.f", 50.0),
		panelRow("spc", "command", ied, "BayACTRL/GGIO1.SPCSO1", nil),
		panelRow("dpc", "command", ied, "BayACTRL/XCBR1.Pos", nil),
		panelRow("ro", "status", ied, "BayACTRL/XCBR1.Pos.Oper.ctlVal", true), // read-only in the IED
	}
}

// panelRun starts BayA with a panel on it (or on BayB, the real device, when real is set).
func panelRun(t *testing.T, portBase int64, real bool) (*Coordinator, *MemoryBus, string) {
	t.Helper()
	folder := fakeNativeDir(t)
	project, a, b := fakeProject(t, 9, portBase, false)
	target := a
	if real {
		projectIeds(project)[1].Set("role", "real")
		target = b
	}
	rows := panelRows(a)
	if real {
		rows = []any{panelRow("s", "status", b, "BayBCTRL/XCBR1.Loc.stVal", true), panelRow("cmd", "command", b, "BayBCTRL/XCBR1.Pos", nil)}
	}
	nodes := pyval.List(pyval.Map(project, "diagram"), "nodes")
	pyval.Map(project, "diagram").Set("nodes", append(nodes, panelNode(target, rows...)))
	bus := &MemoryBus{}
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: bus, Project: NewMemoryProject(project)})
	t.Cleanup(c.Close)
	return c, bus, a
}

func kinds(bus *MemoryBus) []string {
	var found []string
	for _, event := range bus.Events() {
		if kind := pyval.Str(event, "kind"); strings.HasPrefix(kind, "point_panel_") {
			found = append(found, kind)
		}
	}
	return found
}

func TestPlantStatesIncludePointRows(t *testing.T) {
	p := newPlantStates(&Coordinator{})
	node := panelNode("ied1", panelRows("ied1")...)
	other := pyval.Obj("id", "cb", "data", pyval.Obj("kind", "CB", "state_mapping", nil, "relay_statuses", []any{},
		"points", []any{panelRow("x", "status", "ied1", "LD0/IGNORED.stVal", nil)}))
	p.reset(pyval.Obj("nodes", []any{node, other}, "edges", []any{}))
	if want := []string{"pp:s", "pp:d", "pp:v", "pp:ro"}; !reflect.DeepEqual(p.order, want) {
		t.Fatalf("order = %v, want %v (commands have no value; other kinds are not panels)", p.order, want)
	}
	if got := p.bySource[bindingKey{"ied1", "BayACTRL/GGIO1.AnIn1.mag.f"}]; !reflect.DeepEqual(got, []string{"pp:v"}) {
		t.Fatalf("a point change must republish its row, got %v", got)
	}
}

func TestPointPanelSetWritesTheSimulatedIED(t *testing.T) {
	c, bus, a := panelRun(t, 23100, false)
	if _, err := c.EquipmentTest("pp", "set", true, "s"); err == nil || !strings.Contains(err.Error(), "Run the IEDs") {
		t.Fatalf("a stopped run must say how to proceed, got %v", err)
	}
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	answer, err := c.EquipmentTest("pp", "set", true, "s")
	if err != nil || pyval.Get(answer, "value") != true || pyval.Get(answer, "type") != "status" ||
		pyval.Str(answer, "correlation_id") == "" {
		t.Fatalf("status: %v %v", answer, err)
	}
	if value, _ := c.Value(a, "BayACTRL/XCBR1.Loc.stVal"); value != true {
		t.Fatalf("the IED holds %v", value)
	}
	if _, err = c.EquipmentTest("pp", "set", int64(1), "d"); err != nil {
		t.Fatal(err)
	}
	if value, _ := c.Value(a, "BayACTRL/XCBR1.Pos.stVal"); value != int64(1) {
		t.Fatalf("double point holds %v", value)
	}
	if _, err = c.EquipmentTest("pp", "set", 49.5, "v"); err != nil {
		t.Fatal(err)
	}
	if value, _ := c.Value(a, "BayACTRL/GGIO1.AnIn1.mag.f"); value != 49.5 {
		t.Fatalf("value point holds %v", value)
	}
	want := []string{"point_panel_requested", "point_panel_applied", "point_panel_requested", "point_panel_applied",
		"point_panel_requested", "point_panel_applied"}
	if got := kinds(bus); !reflect.DeepEqual(got, want) {
		t.Fatalf("events %v, want %v", got, want)
	}
}

func TestPointPanelRefusesWhatDoesNotFit(t *testing.T) {
	c, bus, a := panelRun(t, 23200, false)
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		action, value, row any
		want               string
	}{
		{"set", int64(1), "s", "ON or OFF"},
		{"set", int64(4), "d", "0 (intermediate), 1 (open), 2 (closed) or 3 (bad)"},
		{"set", true, "v", "enter a number"},
		{"set", "49", "v", "enter a number"},
		{"set", true, "ro", "read-only"},
		{"set", true, "spc", "is a command"},
		{"send", true, "s", "is not a command"},
		{"send", int64(1), "spc", "true or false"},
		{"set", true, "missing", "no such point"},
		{"set", true, nil, "no such point"},
		{"burn", true, "s", "Unknown point action"},
	}
	for _, item := range cases {
		_, err := c.EquipmentTest("pp", item.action, item.value, item.row)
		if err == nil || !strings.Contains(err.Error(), item.want) {
			t.Errorf("%v %v %v: want %q, got %v", item.action, item.value, item.row, item.want, err)
		}
	}
	if len(kinds(bus)) == 0 {
		t.Error("a refused request after the point was found is logged as failed")
	}
	if value, _ := c.Value(a, "BayACTRL/XCBR1.Loc.stVal"); value == true {
		t.Error("a refused request must not write")
	}
}

func TestPointPanelHealthyReportsEveryPoint(t *testing.T) {
	c, bus, a := panelRun(t, 23300, false)
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	answer, err := c.EquipmentTest("pp", "healthy", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pyval.Get(answer, "requested") != int64(4) || pyval.Get(answer, "applied") != int64(3) ||
		pyval.Get(answer, "failed") != int64(1) {
		t.Fatalf("summary %v", answer)
	}
	byRow := map[string]*pyjson.Object{}
	for _, item := range pyval.List(answer, "results") {
		byRow[pyval.Str(item.(*pyjson.Object), "row")] = item.(*pyjson.Object)
	}
	if pyval.Get(byRow["s"], "ok") != true || pyval.Get(byRow["d"], "value") != int64(2) || pyval.Get(byRow["v"], "value") != 50.0 {
		t.Fatalf("results %v", byRow)
	}
	if failed := byRow["ro"]; pyval.Get(failed, "ok") != false || !strings.Contains(pyval.Str(failed, "error"), "read-only") {
		t.Fatalf("the read-only point reports why: %v", failed)
	}
	for ref, want := range map[string]any{"BayACTRL/XCBR1.Loc.stVal": true, "BayACTRL/XCBR1.Pos.stVal": int64(2)} {
		if value, _ := c.Value(a, ref); value != want {
			t.Errorf("%s = %v, want %v", ref, value, want)
		}
	}
	if got := kinds(bus); !reflect.DeepEqual(got, []string{"point_panel_requested", "point_panel_applied"}) {
		t.Fatalf("one request and one result are logged for the batch, got %v", got)
	}
}

func TestPointPanelSendUsesTheCommandPath(t *testing.T) {
	c, bus, a := panelRun(t, 23400, false)
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	answer, err := c.EquipmentTest("pp", "send", true, "dpc")
	if err != nil {
		t.Fatal(err)
	}
	if command := pyval.Map(answer, "command"); pyval.Get(command, "ok") != true || pyval.Str(answer, "ref") != "BayACTRL/XCBR1.Pos" {
		t.Fatalf("answer %v", answer)
	}
	if got := kinds(bus); !reflect.DeepEqual(got, []string{"point_panel_requested", "point_panel_applied"}) {
		t.Fatalf("events %v", got)
	}
}

func TestPointPanelNeverChangesARealDevice(t *testing.T) {
	c, _, _ := panelRun(t, 23500, true)
	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	for _, item := range [][3]any{{"set", true, "s"}, {"send", true, "cmd"}, {"healthy", nil, nil}} {
		_, err := c.EquipmentTest("pp", item[0], item[1], item[2])
		if err == nil || !strings.Contains(err.Error(), "real device") {
			t.Errorf("%v on a real IED: want the real-device refusal, got %v", item[0], err)
		}
	}
}
