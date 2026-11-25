package project

import (
	"strings"
	"testing"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Re-import (ADR-0150): the IED keeps its id, settings and every link its new file still has.

func reimportProject() *pyjson.Object {
	ieds := testIEDs()
	ieds[0].Set("role", "real").Set("live", o("ip", "172.26.0.40", "port", int64(102), "allow_commands", true)).
		Set("lab_ip", "192.168.236.51")
	list := []any{}
	for _, ied := range ieds {
		list = append(list, ied)
	}
	gateway := o("points", o("iec104", []any{o("ied", "IED0", "ref", signalRef(8), "kind", "double", "address", int64(1)),
		o("ied", "IED0", "ref", signalRef(1), "kind", "status", "address", int64(2))}))
	return o("ieds", list, "diagram", testDiagram(), "sources", []any{}, "test_panel", []any{}, "scenarios", []any{},
		"gateway", gateway)
}

// newFile is IED0 as a newer CID: signal 8 is gone, signal 12 became BOOLEAN, CSWI1 is gone,
// and a signal was added.
func newFile(project *pyjson.Object) *pyjson.Object {
	old, _ := FindIED(project, "ied0")
	signals := []any{}
	for _, item := range items(pyval.Lookup(old, "signals", nil)) {
		signal := pyval.CopyObject(object(item))
		switch text(signal, "ref") {
		case signalRef(8):
			continue
		case signalRef(12):
			signal.Set("kind", "BOOLEAN")
		}
		signals = append(signals, signal)
	}
	signals = append(signals, o("ref", "LD0/GGIO99.Ind999.stVal", "kind", "BOOLEAN", "fc", "ST", "writable", true))
	controls := []any{}
	for _, item := range items(pyval.Lookup(old, "controls", nil)) {
		if text(object(item), "ref") != "LD0/CSWI1.Pos" {
			controls = append(controls, item)
		}
	}
	return o("id", "fresh-id", "name", "IED0", "filename", "IED0_v2.cid", "signals", signals, "controls", controls,
		"ip", "", "port", int64(102), "state", "stopped")
}

func equipmentData(t *testing.T, project *pyjson.Object, label string) *pyjson.Object {
	t.Helper()
	for _, item := range diagramNodes(project) {
		if data := object(pyval.Lookup(object(item), "data", nil)); text(data, "label") == label {
			return data
		}
	}
	t.Fatalf("no equipment %s", label)
	return nil
}

func TestReimportKeepsIdentitySettingsAndLinks(t *testing.T) {
	project := reimportProject()
	before := dumps(t, project)
	plan, err := Reimport(project, "ied0", newFile(project))
	if err != nil {
		t.Fatal(err)
	}
	if dumps(t, project) != before {
		t.Fatal("planning must not change the current project")
	}
	ied, _ := FindIED(plan.Project, "ied0")
	if text(ied, "filename") != "IED0_v2.cid" || text(ied, "role") != "real" || text(ied, "lab_ip") != "192.168.236.51" ||
		pyval.Lookup(object(pyval.Lookup(ied, "live", nil)), "allow_commands", nil) != true {
		t.Fatalf("id and settings must stay, the file changes: %s", dumps(t, ied))
	}
	// Equipment of IED0 (n0 CB, n1 DS, n3 TRIP_RELAY) loses only what the new file lacks.
	if cb := equipmentData(t, plan.Project, "CB n0"); pyval.Lookup(cb, "state_mapping", nil) != nil ||
		pyval.Lookup(cb, "control_binding", nil) == nil {
		t.Fatalf("CB: position (signal 8 gone) removed, command kept: %s", dumps(t, cb))
	}
	if ds := equipmentData(t, plan.Project, "DS n1"); pyval.Lookup(ds, "state_mapping", nil) != nil {
		t.Fatalf("DS: a position whose type changed is removed: %s", dumps(t, ds))
	}
	if trip := equipmentData(t, plan.Project, "TRIP_RELAY n3"); pyval.Lookup(trip, "state_mapping", nil) == nil ||
		pyval.Lookup(trip, "control_binding", nil) != nil {
		t.Fatalf("TRIP_RELAY: state kept, command (CSWI1 gone) removed: %s", dumps(t, trip))
	}
	// Other IEDs' equipment is untouched.
	for _, item := range diagramNodes(plan.Project) {
		data := object(pyval.Lookup(object(item), "data", nil))
		if binding := object(pyval.Lookup(data, "control_binding", nil)); text(binding, "ied_id") == "ied1" && text(data, "kind") == "CB" &&
			pyval.Lookup(data, "state_mapping", nil) == nil {
			t.Fatalf("IED1's equipment must keep its links: %s", dumps(t, data))
		}
	}
	words := []string{}
	for _, item := range plan.Lost {
		words = append(words, item.Where+" | "+item.Ref+" | "+item.Why)
	}
	joined := strings.Join(words, "\n")
	for _, want := range []string{"SLD CB n0 position | " + signalRef(8) + " | not in the new file",
		"SLD DS n1 position | " + signalRef(12) + " | its type changed from Dbpos to BOOLEAN",
		"SLD TRIP_RELAY n3 command | LD0/CSWI1.Pos | not in the new file"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing lost line %q in\n%s", want, joined)
		}
	}
	if len(plan.Lost) != 3 || plan.Kept == 0 {
		t.Fatalf("exactly three links lost, others kept (%d): %s", plan.Kept, joined)
	}
	if len(plan.Gateway) != 1 || plan.Gateway[0] != "iec104: "+signalRef(8) {
		t.Fatalf("gateway points whose reference is gone are named: %v", plan.Gateway)
	}
	if _, err := ValidateDiagram(pyval.Lookup(plan.Project, "diagram", nil), iedsOf(plan.Project)); err != nil {
		t.Fatalf("the result opens: %v", err)
	}
}

func TestReimportOfTheSameFileChangesNothing(t *testing.T) {
	project := reimportProject()
	old, _ := FindIED(project, "ied0")
	same := pyval.CopyObject(old).Set("id", "fresh")
	plan, err := Reimport(project, "ied0", same)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Lost) != 0 || len(plan.Gateway) != 0 {
		t.Fatalf("nothing is lost: %v %v", plan.Lost, plan.Gateway)
	}
	if dumps(t, pyval.Lookup(plan.Project, "diagram", nil)) != dumps(t, pyval.Lookup(project, "diagram", nil)) {
		t.Fatal("the same file keeps the diagram byte for byte")
	}
}

func TestReimportRefusesAnotherIED(t *testing.T) {
	project := reimportProject()
	other := newFile(project).Set("name", "IED7")
	if _, err := Reimport(project, "ied0", other); err == nil || !strings.Contains(studioMessage(err), "is IED7, not IED0") {
		t.Fatalf("another IED's file is refused: %v", err)
	}
}

func iedsOf(project *pyjson.Object) []*pyjson.Object {
	var result []*pyjson.Object
	for _, item := range items(pyval.Lookup(project, "ieds", nil)) {
		result = append(result, object(item))
	}
	return result
}
