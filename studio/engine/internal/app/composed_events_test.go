package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/testutil"
)

// runtimeTestdata is the runtime lane's parity scenario folder.
var runtimeTestdata = filepath.Join("..", "runtime", "testdata")

// openScenarioProject restores the runtime parity project (IEDs at portBase+1, +2).
func openScenarioProject(t *testing.T, engine *Engine, portBase int) []string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(runtimeTestdata, "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pyjson.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	tree := decoded.(*pyjson.Object)
	var ids []string
	for index, item := range field(tree, "ieds").([]any) {
		item.(*pyjson.Object).Set("port", int64(portBase+index+1))
		ids = append(ids, field(item, "id").(string))
	}
	payload, err := archive.NewEncoder().Encode(project.Compact(tree))
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.Projects.Restore(payload, true); err != nil {
		t.Fatal(err)
	}
	return ids
}

// TestComposedEventsMatchPython runs the runtime lane's scripted scenario (record_python.py)
// through the composed HTTP routes and compares the events with compare_events.py.
func TestComposedEventsMatchPython(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	nativeDir := muxNative(t)
	if python == "" {
		t.Skip("set STUDIO_PYTHON to record the Python events")
	}
	folder := testutil.TempDir(t)
	pythonOut, goOut := filepath.Join(folder, "python.json"), filepath.Join(folder, "go.json")
	portBase := freePortBase(t, 2, scenarioHosts...)
	record := exec.Command(python, "-I", filepath.Join(runtimeTestdata, "record_python.py"), nativeDir,
		itoa(portBase), pythonOut)
	if output, err := record.CombinedOutput(); err != nil {
		t.Fatalf("record_python.py: %v %s", err, output)
	}
	engine := composeAt(t, testutil.TempDir(t), nativeDir, pythonVerdicts(t, filepath.Join(runtimeTestdata, "project.json")))
	ids := openScenarioProject(t, engine, portBase)
	cursor := engine.Bus.Sequence()
	wait := func(seconds float64) { time.Sleep(time.Duration(seconds * float64(time.Second))) }
	post := func(path string, body any) { mustCall(t, engine, "POST", path, body) }
	post("/api/v1/runtime/start", nil)
	wait(0.5)
	post("/api/v1/equipment/cb1/operate", obj("value", true))
	wait(0.3) // the IED reports the new position at the worker's pace: let it arrive first
	post("/api/v1/equipment/ds1/operate", obj("action", "open"))
	wait(0.3)
	post("/api/v1/signals/update", obj("ied_id", ids[0], "ref", "BayACTRL/GGIO1.AnIn2.mag.f", "value", 42.5))
	post("/api/v1/signals/pulse", obj("ied_id", ids[0], "ref", "BayACTRL/XCBR1.Loc.stVal", "active_value", true,
		"duration_ms", int64(300)))
	wait(0.6)
	post("/api/v1/scenarios/s1/start", nil)
	wait(0.9)
	post("/api/v1/equipment/es1/test", obj("action", "closed"))
	post("/api/v1/equipment/pb1/operate", obj("action", "press"))
	post("/api/v1/runtime/ieds/"+ids[1]+"/stop", nil)
	post("/api/v1/runtime/ieds/"+ids[1]+"/start", nil)
	wait(0.3)
	post("/api/v1/runtime/stop", nil)
	if err := engine.Close(); err != nil { // record_python.py closes its coordinator too
		t.Fatal(err)
	}
	var events []any
	for _, event := range engine.Bus.After(cursor, 0, nil) {
		if event.Kind == "project_changed" { // Python's recorder had no on_change
			continue
		}
		decoded, _ := pyjson.Decode(event.Data)
		events = append(events, decoded)
	}
	encoded, _ := pyjson.Dumps(events)
	if err := os.WriteFile(goOut, encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(python, "-I", filepath.Join(runtimeTestdata, "compare_events.py"), pythonOut, goOut).CombinedOutput()
	t.Logf("compare_events.py:\n%s", output)
	if err != nil {
		t.Fatalf("events differ from Python: %v", err)
	}
}
