// Package goldentest loads golden cases written by the Python engine and compares Go
// output with them byte for byte. Used by tests of the acquisition packages only.
package goldentest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// Case is one recorded call: its arguments and the exact bytes Python produced.
type Case struct {
	Name string
	Args []any
	Want string
}

// Arg returns argument index as an object.
func (c Case) Obj(index int) *pyjson.Object { return pyv.Obj(c.Args[index]) }

// Load reads internal/acquisition/testdata/generated/<file>.
func Load(t *testing.T, file string) []Case {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(Dir(), file))
	if err != nil {
		t.Fatalf("golden %s: %v (run the Python generator in internal/acquisition/testdata)", file, err)
	}
	decoded, err := pyjson.Decode(data)
	if err != nil {
		t.Fatalf("golden %s: %v", file, err)
	}
	var cases []Case
	for _, item := range pyv.Objects(pyv.List(decoded)) {
		cases = append(cases, Case{Name: pyv.GetStr(item, "name"), Args: pyv.GetList(item, "args"),
			Want: pyv.GetStr(item, "want")})
	}
	if len(cases) == 0 {
		t.Fatalf("golden %s has no cases", file)
	}
	return cases
}

// Dir is the shared golden folder internal/acquisition/testdata/generated.
func Dir() string {
	wd, _ := os.Getwd()
	for dir := wd; ; dir = filepath.Dir(dir) {
		if filepath.Base(dir) == "acquisition" {
			return filepath.Join(dir, "testdata", "generated")
		}
		if strings.HasSuffix(dir, "internal") || filepath.Dir(dir) == dir {
			return filepath.Join(dir, "acquisition", "testdata", "generated")
		}
	}
}

// Flask compares pyjson.Flask(got) with the recorded jsonify bytes.
func Flask(t *testing.T, c Case, got any) {
	t.Helper()
	body, err := pyjson.Flask(got)
	if err != nil {
		t.Fatalf("%s: encode: %v", c.Name, err)
	}
	Text(t, c, string(body))
}

// Text compares a text result (CSV, HTML, or already encoded JSON) with the recording.
func Text(t *testing.T, c Case, got string) {
	t.Helper()
	if got != c.Want {
		t.Errorf("%s: differs from Python\n got: %s\nwant: %s", c.Name, clip(got), clip(c.Want))
	}
}

func clip(text string) string {
	if len(text) > 3000 {
		return text[:3000] + "..."
	}
	return text
}
