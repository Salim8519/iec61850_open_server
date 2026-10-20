package selftest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/licensing"
	"studio-engine/internal/testutil"
)

// nativeFolder is STUDIO_NATIVE_BIN, else the source tree's native build; the test skips
// when the IED worker or the probe is missing there.
func nativeFolder(t *testing.T) string {
	t.Helper()
	folder := os.Getenv("STUDIO_NATIVE_BIN")
	if folder == "" {
		folder = filepath.Join("..", "..", "..", "native", "build", "Release")
	}
	folder, _ = filepath.Abs(folder)
	for _, name := range []string{"studio_ied.exe", "studio_probe.exe"} {
		if _, err := os.Stat(filepath.Join(folder, name)); err != nil {
			t.Skipf("%s is not built in %s", name, folder)
		}
	}
	return folder
}

// testOptions are the development Options: Python's verdicts stand in for studio_xsd.exe.
func testOptions(t *testing.T, data, native string) Options {
	schemas, _ := filepath.Abs(filepath.Join("..", "..", "..", "schemas"))
	seed, _ := filepath.Abs(filepath.Join("testdata", "xsd_seed.json"))
	return Options{DataDir: data, NativeDir: native, SchemasDir: schemas, XSDSeed: []string{seed},
		Version: "0.0.0+dev", LicenseHome: testutil.TempDir(t), Logf: t.Logf}
}

// pythonResult is the JSON line Python's check() printed: run now against the same native
// folder when STUDIO_PYTHON is set, else the recorded testdata/python_check.json.
func pythonResult(t *testing.T, native string) string {
	t.Helper()
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		recorded, err := os.ReadFile(filepath.Join("testdata", "python_check.json"))
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(recorded))
	}
	scratch := testutil.TempDir(t)
	backend, _ := filepath.Abs(filepath.Join("..", "..", "..", "backend"))
	result := filepath.Join(scratch, "result.json")
	if output, err := exec.Command(python, "-I", filepath.Join("testdata", "python_check.py"), "check", backend,
		native, filepath.Join(scratch, "data"), filepath.Join(scratch, "seed.json"), result).CombinedOutput(); err != nil {
		t.Fatalf("python_check.py check: %v %s", err, output)
	}
	text, err := os.ReadFile(result)
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func TestCheckPrintsPythonJSON(t *testing.T) {
	native := nativeFolder(t)
	data := filepath.Join(testutil.TempDir(t), "data")
	var stdout, stderr bytes.Buffer
	if code := Run(testOptions(t, data, native), &stdout, &stderr); code != 0 {
		t.Fatalf("self-test exit %d: %s", code, stderr.String())
	}
	if got, want := stdout.String(), pythonResult(t, native)+"\n"; got != want {
		t.Fatalf("Go printed\n%s\nPython printed\n%s", got, want)
	}
	assertEmpty(t, data)
}

func TestCheckRefusesUnlicensedEnforcedEngine(t *testing.T) {
	data := filepath.Join(testutil.TempDir(t), "data")
	options := testOptions(t, data, testutil.TempDir(t))
	options.Enforce = true
	_, err := Check(options)
	want := "The self-test runs the licensed engine, so activate ScadaBench on this computer first. " +
		licensing.NoLicense
	if err == nil || err.Error() != want {
		t.Fatalf("error %v, want %q", err, want)
	}
	assertEmpty(t, data)
}

func TestCheckFailsWithoutWorkersAndCleansUp(t *testing.T) {
	data := filepath.Join(testutil.TempDir(t), "data")
	var stdout, stderr bytes.Buffer
	if code := Run(testOptions(t, data, testutil.TempDir(t)), &stdout, &stderr); code != 1 || stdout.Len() != 0 {
		t.Fatalf("exit %d, stdout %q", code, stdout.String())
	}
	if got := strings.TrimSpace(stderr.String()); got != "Native IED engine is not built yet" {
		t.Fatalf("stderr %q", got)
	}
	assertEmpty(t, data)
}

// assertEmpty checks the temporary studio-check-* folder was removed from the data folder.
func assertEmpty(t *testing.T, data string) {
	t.Helper()
	entries, err := os.ReadDir(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		t.Errorf("left behind in the data folder: %s", entry.Name())
	}
}
