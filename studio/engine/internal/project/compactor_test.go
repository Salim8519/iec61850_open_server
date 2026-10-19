package project

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestCompactorMatchesCompact: the remembering compactor gives Compact's bytes for new,
// shared and edited IED lists, and reuses the stored values when the list is unchanged.
func TestCompactorMatchesCompact(t *testing.T) {
	ied := func(id, source string) *pyjson.Object {
		return o("id", id, "name", id, "filename", id+".scd", "source", source, "signals", []any{},
			"ip", "127.0.0.2", "port", int64(102), "role", "simulated")
	}
	big := strings.Repeat("<SCL/>", 1000)
	lists := [][]any{{}, {ied("a", big)}, {ied("a", big), ied("b", big)}, {ied("b", "x"), ied("a", big)}}
	compactor := NewCompactor()
	for round := 0; round < 3; round++ {
		for _, list := range lists {
			project := o("id", "p", "name", "n", "ieds", list, "diagram", o("nodes", []any{}),
				"migration_notice", "x", "format", int64(1))
			got, _ := pyjson.Dumps(compactor.Compact(project))
			want, _ := pyjson.Dumps(Compact(project))
			if string(got) != string(want) {
				t.Fatalf("round %d: %s\nwant %s", round, got, want)
			}
		}
	}
	project := o("ieds", lists[2])
	first, second := compactor.Compact(project), compactor.Compact(o("ieds", lists[2], "name", "y"))
	a, _ := first.Get("scl_sources")
	b, _ := second.Get("scl_sources")
	if a != b {
		t.Error("an unchanged IED list should reuse the stored source table")
	}
}
