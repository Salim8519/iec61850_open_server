package signal

import (
	"math"
	"os"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// TestCSVTracesMatchCPython replays testdata/csv_vectors.py: the stored spec (json.dumps
// bytes) or the ValueError text, and the played values at several times.
func TestCSVTracesMatchCPython(t *testing.T) {
	raw, err := os.ReadFile("testdata/csv_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pyjson.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	for i, item := range decoded.([]any) {
		row := item.([]any)
		kind, spec, want := row[0].(string), row[1].(*pyjson.Object), row[2].(string)
		result, err := ValidateTrace(Signal{Kind: kind, FC: "MX"}, spec)
		var got string
		if err != nil {
			got = "ValueError: " + err.Error()
		} else {
			encoded, _ := pyjson.Dumps(result)
			got = string(encoded)
		}
		if got != want {
			t.Errorf("row %d (%s %q):\n go     %s\n python %s", i, kind, pyval.Get(spec, "csv_text"), got, want)
			continue
		}
		for _, sample := range row[3].([]any) {
			pair := sample.([]any)
			elapsed, _ := pyval.Float(pair[0])
			value := TraceValue(result, elapsed)
			if !pyval.Equal(value, pair[1]) || pyval.TypeName(value) != pyval.TypeName(pair[1]) {
				expected, _ := pyval.Float(pair[1])
				actual, _ := pyval.Float(value)
				if !(math.IsNaN(expected) && math.IsNaN(actual)) {
					t.Errorf("row %d at %v: go %v python %v", i, elapsed, value, pair[1])
				}
			}
		}
	}
}
