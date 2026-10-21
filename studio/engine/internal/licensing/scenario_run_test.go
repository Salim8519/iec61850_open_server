package licensing

import (
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// snapshotFiles is make_scenarios.snapshot: every file under root except the native
// folder, as base64, keyed by its slash path.
func snapshotFiles(t *testing.T, root string) map[string]string {
	files := map[string]string{}
	filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, _ := filepath.Rel(root, path)
		relative = filepath.ToSlash(relative)
		if !strings.HasPrefix(relative, "native/") {
			data, _ := os.ReadFile(path)
			files[relative] = b64Text(data)
		}
		return nil
	})
	return files
}

// sameFiles compares snapshots; off Windows, CRLF from Python's text mode is not expected.
func sameFiles(t *testing.T, got, want map[string]string) string {
	if len(got) != len(want) {
		return fmt.Sprintf("files %v, want %v", keysOf(got), keysOf(want))
	}
	for name, data := range want {
		expected, actual := string(fromB64(t, data)), string(fromB64(t, got[name]))
		if runtime.GOOS != "windows" {
			expected = strings.ReplaceAll(expected, "\r\n", "\n")
		}
		if _, found := got[name]; !found || actual != expected {
			return fmt.Sprintf("%s = %q, want %q", name, actual, expected)
		}
	}
	return ""
}

func keysOf(files map[string]string) []string {
	names := []string{}
	for name := range files {
		names = append(names, name)
	}
	return names
}

func applyFileStep(t *testing.T, root string, step scenarioStep) {
	target := filepath.Join(root, filepath.FromSlash(step.Path))
	os.MkdirAll(filepath.Dir(target), 0o777)
	os.Remove(target)
	if step.Op == "remove" {
		return
	}
	data := []byte(nil)
	if step.Data != nil {
		data = []byte(*step.Data)
	} else {
		data, _ = hex.DecodeString(step.Hex)
	}
	if err := os.WriteFile(target, data, 0o666); err != nil {
		t.Fatal(err)
	}
}

// TestScenariosMatchPython drives the Go service through every recorded Python run and
// compares HTTP answers, client calls and file bytes.
func TestScenariosMatchPython(t *testing.T) {
	var recorded struct {
		AppVersion string     `json:"app_version"`
		Scenarios  []scenario `json:"scenarios"`
	}
	readVectors(t, "scenarios.json", &recorded)
	t.Setenv("STUDIO_LICENSE_CERT_PATH", "")
	for _, sc := range recorded.Scenarios {
		t.Run(sc.Name, func(t *testing.T) {
			root := scenarioDir(t)
			clock, calls := 0.0, [][]any{}
			service := newScenarioService(t, sc, root, &clock, &calls)
			for index, step := range sc.Steps {
				if step.Now != nil {
					clock = *step.Now
				}
				if step.Op == "write" || step.Op == "remove" {
					applyFileStep(t, root, step)
					continue
				}
				calls = calls[:0]
				want := sc.Results[index]
				got, err := runScenarioStep(t, service, step, recorded.AppVersion)
				switch {
				case want.Crash != "" && err == nil:
					t.Errorf("step %d %s: no error, Python raised %s", index, step.Op, want.Crash)
				case want.Crash == "" && err != nil:
					t.Errorf("step %d %s: error %v", index, step.Op, err)
				case want.Crash == "" && (got.HTTP != want.HTTP || got.Body != want.Body ||
					string(got.Value) != string(want.Value)):
					t.Errorf("step %d %s:\n got %d %s %s\nwant %d %s %s", index, step.Op, got.HTTP, got.Body,
						got.Value, want.HTTP, want.Body, want.Value)
				}
				if fmt.Sprint(calls) != fmt.Sprint(want.Calls) {
					t.Errorf("step %d %s: calls %v, want %v", index, step.Op, calls, want.Calls)
				}
				if problem := sameFiles(t, snapshotFiles(t, root), want.Files); problem != "" {
					t.Errorf("step %d %s: %s", index, step.Op, problem)
				}
			}
		})
	}
}

// scenarioDir is a temporary folder whose removal is retried: on Windows a virus scanner or
// the search indexer may hold a just-written license file for a moment.
func scenarioDir(t *testing.T) string {
	root, err := os.MkdirTemp("", "license-scenario-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for attempt := 0; attempt < 50; attempt++ {
			if os.RemoveAll(root) == nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("could not remove %s", root)
	})
	return root
}
