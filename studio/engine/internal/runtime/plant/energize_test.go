package plant

import (
	"os"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// TestEnergizationMatchesCPython compares the energization event payload ({"nodes", "edges",
// "hints"} as json.dumps writes it) with CPython for every vector of energize_vectors.py.
func TestEnergizationMatchesCPython(t *testing.T) {
	raw, err := os.ReadFile("testdata/energize_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := pyjson.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	diagrams := pyval.List(decoded.(*pyjson.Object), "diagrams")
	rows := pyval.List(decoded.(*pyjson.Object), "rows")
	for i, item := range rows {
		row := item.([]any)
		diagram := diagrams[row[0].(int64)].(*pyjson.Object)
		positions := map[string]any{}
		object := row[1].(*pyjson.Object)
		for _, key := range object.Keys() {
			positions[key], _ = object.Get(key)
		}
		sections := NewSections(diagram, positions)
		nodes, edges := sections.Describe()
		got, _ := pyjson.Dumps(pyval.Obj("nodes", nodes, "edges", edges, "hints", sections.SwitchingHints()))
		if string(got) != row[2].(string) {
			t.Fatalf("row %d positions %v:\n go     %s\n python %s", i, positions, got, row[2])
		}
	}
	t.Logf("%d energisation vectors identical", len(rows))
}

func TestCommandAndFeedbackRules(t *testing.T) {
	breaker := pyval.Obj("kind", "CB")
	if action, wire, err := SemanticCommand(breaker, "", true); err != nil || action != "close" || !wire {
		t.Fatalf("CB true: %v %v %v", action, wire, err)
	}
	if _, _, err := SemanticCommand(breaker, "open", true); err == nil || err.Error() != "Send an equipment action or a raw value, not both" {
		t.Fatalf("both: %v", err)
	}
	trip := pyval.Obj("kind", "TRIP_RELAY", "trip_command_polarity", "operate_true")
	if action, wire, _ := SemanticCommand(trip, "reset", nil); action != "reset" || wire {
		t.Fatalf("trip reset: %v %v", action, wire)
	}
	mapping := pyval.Obj("mode", "contacts_to_dbpos", "open", pyval.Obj("ied_id", "a", "ref", "o"),
		"closed", pyval.Obj("ied_id", "a", "ref", "c"))
	targets, _ := FeedbackTargets(breaker, mapping, "close")
	if len(targets) != 2 || targets[0].Value != false || targets[1].Value != true {
		t.Fatalf("contacts: %+v", targets)
	}
	values := map[Binding]any{{"a", "o"}: false, {"a", "c"}: true}
	if state := EvaluateState(mapping, func(b Binding) any { return values[b] }); state != int64(2) {
		t.Fatalf("contacts state %v", state)
	}
}
