package runtime

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// nativeDir is the folder of the untracked native workers (STUDIO_NATIVE_BIN overrides).
func nativeDir(t *testing.T) string {
	t.Helper()
	folder := os.Getenv("STUDIO_NATIVE_BIN")
	if folder == "" {
		folder = filepath.Join("..", "..", "..", "native", "build", "Release")
	}
	folder, _ = filepath.Abs(folder)
	if _, err := os.Stat(filepath.Join(folder, "studio_ied.exe")); err != nil {
		t.Skipf("native build required: %s", folder)
	}
	return folder
}

// loadProject reads the synthetic template project and moves its IEDs to portBase+1, +2.
func loadProject(t testing.TB, portBase int64) *pyjson.Object {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "project.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pyjson.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	project := decoded.(*pyjson.Object)
	for i, ied := range projectIeds(project) {
		ied.Set("port", portBase+int64(i)+1)
	}
	return project
}

// scriptStep is one action of the scripted parity scenario (same list in testdata/record_python.py).
type scriptStep struct {
	name string
	run  func(c *Coordinator, ids []string) error
}

func waitFor(seconds float64) func(*Coordinator, []string) error {
	return func(*Coordinator, []string) error {
		time.Sleep(time.Duration(seconds * float64(time.Second)))
		return nil
	}
}

// parityScript is the scripted scenario run by both engines.
var parityScript = []scriptStep{
	{"start", func(c *Coordinator, _ []string) error { return c.Start(nil) }},
	{"wait", waitFor(0.5)},
	{"operate cb1 close", func(c *Coordinator, _ []string) error { _, err := c.Operate("cb1", true, ""); return err }},
	{"wait", waitFor(0.3)}, // the IED reports the new position at the worker's pace: let it arrive first
	{"operate ds1 open", func(c *Coordinator, _ []string) error { _, err := c.Operate("ds1", nil, "open"); return err }},
	{"wait", waitFor(0.3)},
	{"manual AnIn2", func(c *Coordinator, ids []string) error {
		_, err := c.Update(ids[0], "BayACTRL/GGIO1.AnIn2.mag.f", 42.5, nil, "manual")
		return err
	}},
	{"pulse Loc", func(c *Coordinator, ids []string) error {
		_, err := c.Pulse(ids[0], "BayACTRL/XCBR1.Loc.stVal", true, int64(300))
		return err
	}},
	{"wait", waitFor(0.6)},
	{"scenario s1", func(c *Coordinator, _ []string) error { return c.StartScenario("s1") }},
	{"wait", waitFor(0.9)},
	{"test es1 closed", func(c *Coordinator, _ []string) error {
		_, err := c.EquipmentTest("es1", "closed", nil, nil)
		return err
	}},
	{"press pb1", func(c *Coordinator, _ []string) error { _, err := c.Operate("pb1", nil, "press"); return err }},
	{"stop BayB", func(c *Coordinator, ids []string) error { return c.Stop([]string{ids[1]}) }},
	{"start BayB", func(c *Coordinator, ids []string) error { return c.Start([]string{ids[1]}) }},
	{"wait", waitFor(0.3)},
	{"stop", func(c *Coordinator, _ []string) error { return c.Stop(nil) }},
}

func runScript(t *testing.T, portBase int64) ([]*pyjson.Object, *MemoryProject) {
	requireMux(t, nativeDir(t))
	project := NewMemoryProject(loadProject(t, portBase))
	bus := &MemoryBus{}
	c := New(Options{NativeDir: nativeDir(t), DataDir: testutil.TempDir(t), Bus: bus, Project: project})
	defer func() { c.Close() }()
	var ids []string
	for _, ied := range projectIeds(project.Current()) {
		ids = append(ids, pyval.Str(ied, "id"))
	}
	for _, step := range parityScript {
		started := time.Now()
		if err := step.run(c, ids); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		t.Logf("%s: %v", step.name, time.Since(started))
	}
	c.Close() // record_python.py closes the coordinator too (one more runtime event)
	return bus.Events(), project
}

func TestScriptedRunWithRealWorkers(t *testing.T) {
	events, project := runScript(t, 21100)
	counts := map[string]int{}
	for _, event := range events {
		counts[pyval.Str(event, "kind")]++
	}
	t.Logf("event kinds: %v", counts)
	for _, kind := range []string{"runtime", "signal", "equipment_state", "energization", "equipment_feedback_applied",
		"pulse_started", "pulse_completed", "scenario_started", "scenario_step_applied", "scenario_failed",
		"equipment_test", "equipment_button_applied", "ied_runtime", "control_applied"} {
		if counts[kind] == 0 {
			t.Errorf("no %s event", kind)
		}
	}
	if out := os.Getenv("STUDIO_PARITY_OUT"); out != "" {
		encoded, _ := pyjson.Dumps(toAny(events))
		_ = os.WriteFile(out, encoded, 0o644)
	}
	if project.Saves == 0 {
		t.Error("node start positions were not saved")
	}
}

func toAny(events []*pyjson.Object) []any {
	items := make([]any, len(events))
	for i, event := range events {
		items[i] = event
	}
	return items
}
