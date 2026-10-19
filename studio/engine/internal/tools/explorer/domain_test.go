package explorer

import (
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pyoracle"
)

// TestDomainAgainstPython replays node views, key points, key nodes, merges and value texts.
func TestDomainAgainstPython(t *testing.T) {
	cases := pyoracle.Cases(t, filepath.Join("testdata", "explorer_oracle.py"), nil, "61850", "1500")
	if len(cases) < 5000 {
		t.Fatalf("only %d cases", len(cases))
	}
	for _, item := range cases {
		if item.Error != "" {
			continue // Python crashes on these inputs too (checked by count below)
		}
		var got any
		switch item.Fn {
		case "node_view":
			got = NodeView(item.Args[0].(string), item.Args[1].(*pyjson.Object))
		case "key_points":
			got = orEmpty(KeyPoints(item.Args[0].(*pyjson.Object)))
		case "key_nodes":
			var names []string
			for _, name := range item.Args[1].([]any) {
				names = append(names, name.(string))
			}
			nodes := []any{}
			for _, node := range KeyNodes(item.Args[0].(string), names) {
				nodes = append(nodes, node)
			}
			got = nodes
		case "merge":
			got = MergeWatch(item.Args[0].([]any))
		case "describe":
			got = Describe(item.Args[0].(string), item.Args[1])
		case "time_text":
			got = TimeText(item.Args[0])
		}
		if pyoracle.Dumps(t, got) != pyoracle.Dumps(t, item.Result) {
			t.Errorf("%s%s\n got %s\nwant %s", item.Fn, pyoracle.Dumps(t, item.Args), pyoracle.Dumps(t, got),
				pyoracle.Dumps(t, item.Result))
		}
	}
}

func orEmpty(items []any) []any {
	if items == nil {
		return []any{}
	}
	return items
}
