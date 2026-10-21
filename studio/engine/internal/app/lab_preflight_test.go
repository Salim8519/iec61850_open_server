package app

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

// TestLabLookupFailureMatchesPython (gap 7): Run in lab mode while the Windows address
// lookup fails answers like the frozen Python app, where ip_helper.addresses_on raised
// OSError(87): Flask's generic 500 and a "runtime" stopped event carrying the error text.
func TestLabLookupFailureMatchesPython(t *testing.T) {
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		t.Skip("set STUDIO_PYTHON to run the Python side")
	}
	out := filepath.Join(testutil.TempDir(t), "python.json")
	if output, err := exec.Command(python, "-I", filepath.Join("testdata", "lab_preflight_python.py"), out).CombinedOutput(); err != nil {
		t.Fatalf("lab_preflight_python.py: %v %s", err, output)
	}
	raw, _ := os.ReadFile(out)
	want, err := pyjson.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}

	original := addressesOn
	addressesOn = func(string) (map[string]bool, error) {
		return nil, errors.New("[WinError 87] GetAdaptersAddresses failed") // netcfg's text
	}
	defer func() { addressesOn = original }()
	nativeDir := testutil.TempDir(t)
	if err := os.WriteFile(filepath.Join(nativeDir, "studio_ied.exe"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	engine := composeAt(t, testutil.TempDir(t), nativeDir, pythonVerdicts(t, "template:LabBay"))
	defer engine.Close()
	if _, err := engine.Projects.AddTemplate("LabBay"); err != nil {
		t.Fatal(err)
	}
	if _, err := engine.Projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		network := field(draft, "network").(*pyjson.Object)
		network.Set("mode", "lab").Set("adapter_guid", "{0F1E2D3C-4B5A-6978-8796-A5B4C3D2E1F0}")
		return network, nil
	}); err != nil {
		t.Fatal(err)
	}
	cursor := engine.Bus.Sequence()
	status, body := call(t, engine, "POST", "/api/v1/runtime/start", nil)
	events := []any{}
	for _, event := range engine.Bus.After(cursor, 0, nil) {
		if event.Kind != "runtime" {
			continue
		}
		decoded, _ := pyjson.Decode(event.Data)
		object := decoded.(*pyjson.Object)
		object.Delete("id")
		object.Delete("timestamp")
		events = append(events, object)
	}
	got := obj("status", int64(status), "body", body, "events", events, "state", engine.Runtime.State())
	gotText, wantText := sortedJSON(t, got), sortedJSON(t, want)
	t.Logf("python %s\ngo     %s", wantText, gotText)
	if gotText != wantText {
		t.Fatal("lab-mode Run with a failing address lookup differs from Python")
	}
}
