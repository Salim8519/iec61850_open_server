package settings

import (
	"os"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestToolCleanVectorsFromPython compares clean with tool_memory._clean outputs recorded
// by testdata/crosscheck.py (CPython 3.12).
func TestToolCleanVectorsFromPython(t *testing.T) {
	data, err := os.ReadFile("../testdata/fixtures/toolclean_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	value, err := pyjson.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	for index, item := range value.([]any) {
		pair := item.([]any)
		got, _ := pyjson.Dumps(clean(pair[0]))
		want, _ := pyjson.Dumps(pair[1])
		if string(got) != string(want) {
			t.Errorf("vector %d:\n got %s\nwant %s", index, got, want)
		}
	}
}
