package gateway

import (
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

// TestCSVAgainstPython replays gateway_csv: exports of validated and odd stored points and
// imports of exported texts with BOMs, quotes, blank lines, bad rows and size limits.
func TestCSVAgainstPython(t *testing.T) {
	replay(t, "csv_oracle.py", "2000", func(item pyoracle.Case, fixtures map[string][]*pyjson.Object) {
		args := item.Args
		ieds := fixtures[args[0].(string)]
		switch item.Fn {
		case "export_csv":
			got, err := ExportCSV(list(args[1]), ieds)
			expect(t, item, got, err)
		case "import_csv":
			points, problems, err := ImportCSV(args[1].(string), ieds)
			expect(t, item, []any{points, strs(problems)}, err)
		default:
			t.Errorf("unknown oracle function %s", item.Fn)
		}
	})
}
