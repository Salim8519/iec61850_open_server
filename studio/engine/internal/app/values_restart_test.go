package app

import (
	"archive/zip"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

const restartRef = "BayACTRL/GGIO1.AnIn2.mag.f"

// TestRuntimeValuesAcrossRestartMatchPython (gap 4): the frozen Python app keeps a runtime
// value on the project's signal dict for the rest of the session (after Stop and into the
// next Run), but its archive stores only the IED settings and the SCL source, so a restart
// rebuilds the signals from the SCL and shows the initial value. The composed Go server
// must show the same value at every stage and save the same IED keys.
func TestRuntimeValuesAcrossRestartMatchPython(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	nativeDir := muxNative(t)
	if python == "" {
		t.Skip("set STUDIO_PYTHON to run the Python side")
	}
	pythonOut := filepath.Join(testutil.TempDir(t), "python.json")
	portBase := freePortBase(t, 2, scenarioHosts...)
	run := exec.Command(python, "-I", filepath.Join("testdata", "values_restart_python.py"), nativeDir,
		itoa(portBase), pythonOut)
	if output, err := run.CombinedOutput(); err != nil {
		t.Fatalf("values_restart_python.py: %v %s", err, output)
	}
	raw, err := os.ReadFile(pythonOut)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := pyjson.Decode(raw)

	seeds := pythonVerdicts(t, filepath.Join(runtimeTestdata, "project.json"))
	dataDir := testutil.TempDir(t)
	engine := composeAt(t, dataDir, nativeDir, seeds)
	ids := openScenarioProject(t, engine, portBase)
	got := pyjson.NewObject()
	got.Set("initial", shownValue(t, engine, ids[0]))
	mustCall(t, engine, "POST", "/api/v1/runtime/start", nil)
	mustCall(t, engine, "POST", "/api/v1/signals/update", obj("ied_id", ids[0], "ref", restartRef, "value", 42.5))
	mustCall(t, engine, "POST", "/api/v1/runtime/stop", nil)
	got.Set("after_stop", shownValue(t, engine, ids[0]))
	mustCall(t, engine, "POST", "/api/v1/runtime/start", nil)
	got.Set("second_run", shownValue(t, engine, ids[0]))
	mustCall(t, engine, "POST", "/api/v1/runtime/stop", nil)
	mustCall(t, engine, "POST", "/api/v1/project/save", nil)
	if err := engine.Close(); err != nil {
		t.Fatal(err)
	}
	got.Set("archive_ied_keys", savedIEDKeys(t, filepath.Join(dataDir, "autosave.subsim")))
	reopened := composeAt(t, dataDir, nativeDir, seeds)
	defer reopened.Close()
	got.Set("recovery", field(mustCall(t, reopened, "GET", "/api/v1/project", nil), "startup_recovery", "status"))
	got.Set("after_restart", shownValue(t, reopened, ids[0]))

	gotText, wantText := sortedJSON(t, got), sortedJSON(t, want)
	t.Logf("python %s\ngo     %s", wantText, gotText)
	if gotText != wantText {
		t.Fatal("runtime values across Stop, Run and restart differ from Python")
	}
}

// shownValue is the "value" GET /ieds/{id}/signals?ref=restartRef answers.
func shownValue(t *testing.T, engine *Engine, iedID string) any {
	t.Helper()
	answer := mustCall(t, engine, "GET", "/api/v1/ieds/"+iedID+"/signals?ref="+url.QueryEscape(restartRef), nil)
	items, _ := field(answer, "items").([]any)
	if len(items) != 1 {
		t.Fatalf("signals answer for %s: %s", restartRef, pyjsonText(answer))
	}
	return field(items[0], "value")
}

// savedIEDKeys lists the keys of the IED entries in a saved archive (sorted, unique).
func savedIEDKeys(t *testing.T, path string) []any {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	entry, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer entry.Close()
	data, _ := io.ReadAll(entry)
	decoded, err := pyjson.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, ied := range field(decoded, "ieds").([]any) {
		for _, key := range ied.(*pyjson.Object).Keys() {
			seen[key] = true
		}
	}
	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]any, len(keys))
	for i, key := range keys {
		result[i] = key
	}
	return result
}

// sortedJSON is json.dumps(value, sort_keys=True) for the comparison.
func sortedJSON(t *testing.T, value any) string {
	t.Helper()
	text, err := pyjson.Dumps(sortKeys(value))
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

// sortKeys copies value with every object's keys in sorted order.
func sortKeys(value any) any {
	switch typed := value.(type) {
	case *pyjson.Object:
		sorted := pyjson.NewObject()
		keys := append([]string(nil), typed.Keys()...)
		sort.Strings(keys)
		for _, key := range keys {
			item, _ := typed.Get(key)
			sorted.Set(key, sortKeys(item))
		}
		return sorted
	case []any:
		items := make([]any, len(typed))
		for i, item := range typed {
			items[i] = sortKeys(item)
		}
		return items
	}
	return value
}
