// Package pyoracle runs the frozen Python backend as a test oracle: a script under a
// package's testdata/ imports studio_app, evaluates the Python function on synthetic or
// public-template inputs and prints one JSON document per line. Go tests decode those
// lines with pyjson and must reproduce the Python results byte for byte.
//
// Tests skip when no Python with the studio requirements is found; STUDIO_PYTHON points to
// one (the studio venv), as for the pyjson differential test.
package pyoracle

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/nativetest"
)

// Python returns the interpreter to use, or skips the test.
func Python(t testing.TB) string {
	t.Helper()
	candidates := []string{os.Getenv("STUDIO_PYTHON")}
	studio := studioDir()
	candidates = append(candidates, filepath.Join(studio, ".venv", "Scripts", "python.exe"),
		filepath.Join(studio, ".venv", "bin", "python"))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Skip("no Python with the studio requirements; set STUDIO_PYTHON")
	return ""
}

// studioDir is the studio/ folder of this checkout (the frozen backend lives there).
func studioDir() string { return nativetest.StudioDir() }

// Backend is studio/backend, the import root of the frozen Python package.
func Backend() string { return filepath.Join(studioDir(), "backend") }

// Run executes script (a path, usually testdata/<name>.py) with the backend on
// STUDIO_BACKEND, feeds input as stdin and returns every output line decoded by pyjson.
func Run(t testing.TB, script string, input []byte, args ...string) []any {
	t.Helper()
	python := Python(t)
	command := exec.Command(python, append([]string{"-I", script}, args...)...)
	library := filepath.Join(studioDir(), "engine", "internal", "tools", "pyoracle", "python")
	// -I ignores PYTHONPATH, so scripts add STUDIO_ORACLE_LIB to sys.path themselves.
	command.Env = append(os.Environ(), "STUDIO_BACKEND="+Backend(), "STUDIO_ORACLE_LIB="+library)
	command.Stdin = bytes.NewReader(input)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		t.Fatalf("python oracle %s: %v\n%s", script, err, stderr.String())
	}
	var documents []any
	lines := bufio.NewScanner(bytes.NewReader(output))
	lines.Buffer(make([]byte, 1<<20), 1<<28)
	for lines.Scan() {
		if len(bytes.TrimSpace(lines.Bytes())) == 0 {
			continue
		}
		value, err := pyjson.Decode(lines.Bytes())
		if err != nil {
			t.Fatalf("python oracle %s: bad line %q: %v", script, lines.Text(), err)
		}
		documents = append(documents, value)
	}
	return documents
}

// Case is one oracle line {"fn", "args", "result" | "error"} as the scripts print them.
type Case struct {
	Fn     string
	Args   []any
	Result any
	Error  string // "ValueError: text" when Python raised
	Raw    *pyjson.Object
}

// Cases runs script and splits its lines into Cases.
func Cases(t testing.TB, script string, input []byte, args ...string) []Case {
	t.Helper()
	var cases []Case
	for _, document := range Run(t, script, input, args...) {
		object, ok := document.(*pyjson.Object)
		if !ok {
			t.Fatalf("oracle line is not an object: %v", document)
		}
		item := Case{Raw: object}
		if fn, ok := object.Get("fn"); ok {
			item.Fn, _ = fn.(string)
		}
		if values, ok := object.Get("args"); ok {
			item.Args, _ = values.([]any)
		}
		item.Result, _ = object.Get("result")
		if text, ok := object.Get("error"); ok {
			item.Error, _ = text.(string)
		}
		cases = append(cases, item)
	}
	return cases
}

// Dumps encodes value like json.dumps (insertion order), failing the test on error.
func Dumps(t testing.TB, value any) string {
	t.Helper()
	encoded, err := pyjson.Dumps(value)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return string(encoded)
}
