package crosscheck

import (
	"archive/zip"
	"bytes"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"studio-engine/internal/testutil"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/store/jsonfile"
	"studio-engine/internal/store/settings"
)

// Goldens in testdata/fixtures were written by testdata/crosscheck.py with CPython 3.12.

const fixtures = "../testdata/fixtures"

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtures, name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func projectJSON(t *testing.T, payload []byte) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		t.Fatal(err)
	}
	stream, err := reader.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestIndentGoldensFromPython(t *testing.T) {
	inputs, _ := filepath.Glob(filepath.Join(fixtures, "indent", "*.json"))
	if len(inputs) < 20 {
		t.Fatalf("only %d indent goldens", len(inputs))
	}
	for _, input := range inputs {
		data, _ := os.ReadFile(input)
		value, err := pyjson.Decode(data)
		if err != nil {
			t.Fatalf("%s: %v", input, err)
		}
		golden, _ := os.ReadFile(strings.TrimSuffix(input, ".json") + ".golden")
		got, err := jsonfile.Encode(value)
		if err != nil || !bytes.Equal(got, golden) {
			t.Errorf("%s:\n got %q\nwant %q", filepath.Base(input), got, golden)
		}
	}
}

func TestPythonArchiveDecodesAndReencodesIdentically(t *testing.T) {
	want := readFixture(t, "python_archive.json")
	doc, err := archive.Decode(readFixture(t, "python_archive.scadabench"))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := pyjson.Dumps(doc); !bytes.Equal(got, want) {
		t.Fatalf("decoded document differs:\n%s\n%s", got, want)
	}
	encoder := archive.NewEncoder()
	for round := 0; round < 2; round++ { // cold, then from cached segments
		payload, err := encoder.Encode(doc)
		if err != nil {
			t.Fatal(err)
		}
		if got := projectJSON(t, payload); !bytes.Equal(got, want) {
			t.Fatalf("round %d: Go archive JSON differs from Python's", round)
		}
	}
}

func TestNaNMessageMatchesPython(t *testing.T) {
	_, err := archive.Encode(pyjson.NewObject().Set("a", []any{math.NaN()}))
	if want := string(readFixture(t, "nan_message.txt")); err == nil || err.Error() != want {
		t.Fatalf("got %v want %q", err, want)
	}
}

func TestPythonSettingsFilesResaveIdentically(t *testing.T) {
	for _, name := range []string{settings.PreferencesName, settings.ToolSettingsName} {
		original := readFixture(t, name)
		value, ok := jsonfile.Read(filepath.Join(fixtures, name))
		if !ok {
			t.Fatalf("%s unreadable", name)
		}
		if again, _ := jsonfile.Encode(value); !bytes.Equal(again, original) {
			t.Errorf("%s re-saved differently:\n%q\n%q", name, again, original)
		}
	}
	dir := testutil.TempDir(t)
	original := readFixture(t, settings.ToolSettingsName)
	_ = os.WriteFile(filepath.Join(dir, settings.ToolSettingsName), original, 0o666)
	memory := settings.NewToolMemory(dir)
	if encoded, _ := jsonfile.Encode(memory.Get()); !bytes.Equal(encoded, original) {
		t.Fatalf("cleaned Python tool settings differ:\n%q", encoded)
	}
	// The same last update Python made gives the same file.
	if _, err := memory.Update("find", pyjson.NewObject().Set("range", "10.0.0.0/24").
		Set("port", true).Set("choice", "\u00e9")); err != nil {
		t.Fatal(err)
	}
	if written, _ := os.ReadFile(filepath.Join(dir, settings.ToolSettingsName)); !bytes.Equal(written, original) {
		t.Fatalf("Go update wrote\n%q\nwant\n%q", written, original)
	}
}
