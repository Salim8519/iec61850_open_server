package memory

import (
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

// TestAgainstPython replays random stored files and updates through ToolMemory and
// compares every result and the written file bytes (indent=2, CRLF) with Python's.
func TestAgainstPython(t *testing.T) {
	cases := pyoracle.Cases(t, filepath.Join("testdata", "memory_oracle.py"), nil, "61850", "400")
	if len(cases) == 0 {
		t.Fatal("no cases")
	}
	for number, item := range cases {
		folder := t.TempDir()
		if raw, ok := item.Args[0].(string); ok {
			if err := os.WriteFile(filepath.Join(folder, FileName), []byte(raw), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		store := New(folder)
		results := []any{store.Get()}
		for _, op := range item.Args[1].([]any) {
			pair := op.([]any)
			section, err := store.Update(pair[0].(string), pair[1].(*pyjson.Object))
			if err != nil {
				t.Fatal(err)
			}
			results = append(results, section)
		}
		var file any
		if data, err := os.ReadFile(filepath.Join(folder, FileName)); err == nil {
			file = string(data)
		}
		results = append(results, file)
		if got, want := pyoracle.Dumps(t, results), pyoracle.Dumps(t, item.Result); got != want {
			t.Fatalf("case %d %s\n got %s\nwant %s", number, pyoracle.Dumps(t, item.Args), got, want)
		}
	}
}
