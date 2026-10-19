package pycsv

import (
	"path/filepath"
	"testing"

	"studio-engine/internal/tools/pyoracle"
)

// TestAgainstCPython compares random CSV texts with CPython's csv module (reader and writer).
func TestAgainstCPython(t *testing.T) {
	cases := pyoracle.Cases(t, filepath.Join("testdata", "csv_oracle.py"), nil, "61850", "3000")
	if len(cases) == 0 {
		t.Fatal("no oracle cases")
	}
	for _, item := range cases {
		switch item.Fn {
		case "read":
			text, strict := item.Args[0].(string), item.Args[1].(bool)
			rows, err := ReadAll(text, strict)
			if item.Error != "" {
				if err == nil || "Error: "+err.Error() != item.Error {
					t.Errorf("read %q strict=%v: error %v want %s", text, strict, err, item.Error)
				}
				continue
			}
			got := make([]any, len(rows))
			for index, row := range rows {
				cells := make([]any, len(row))
				for column, cell := range row {
					cells[column] = cell
				}
				got[index] = cells
			}
			if err != nil || pyoracle.Dumps(t, got) != pyoracle.Dumps(t, item.Result) {
				t.Errorf("read %q strict=%v: got %s %v want %s", text, strict, pyoracle.Dumps(t, got), err,
					pyoracle.Dumps(t, item.Result))
			}
		case "write":
			writer := NewWriter()
			for _, row := range item.Args[0].([]any) {
				writer.Row(row.([]any)...)
			}
			if writer.String() != item.Result {
				t.Errorf("write %s: got %q want %q", pyoracle.Dumps(t, item.Args), writer.String(), item.Result)
			}
		}
	}
}
