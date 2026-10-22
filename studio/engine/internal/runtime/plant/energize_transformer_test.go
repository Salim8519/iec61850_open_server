package plant

import (
	"testing"

	"studio-engine/internal/runtime/pyval"
)

func transformerDiagram(infeedState string) (nodes, edges []any) {
	node := func(id, kind string, extra ...any) any {
		data := pyval.Obj(append([]any{"kind", kind, "label", id}, extra...)...)
		return pyval.Obj("id", id, "data", data)
	}
	wire := func(id, source, sourceHandle, target, targetHandle string) any {
		return pyval.Obj("id", id, "source", source, "sourceHandle", sourceHandle, "target", target, "targetHandle", targetHandle)
	}
	nodes = []any{node("grid", "INFEED", "infeed_state", infeedState), node("t1", TapTransformerKind),
		node("lv", "JUNCTION"), node("es", "ES")}
	edges = []any{wire("w1", "grid", "south", "t1", "north"), wire("w2", "t1", "south", "lv", "north"),
		wire("w3", "lv", "east", "es", "north")}
	return nodes, edges
}

func TestTransformerPassesPowerToTheLVSide(t *testing.T) {
	nodes, edges := transformerDiagram("live")
	sections := NewSections(pyval.Obj("nodes", nodes, "edges", edges), map[string]any{"es": openPosition})
	if got := sections.State("t1:b"); got != "live" {
		t.Fatalf("LV side: %s", got)
	}
	if got := sections.State("lv"); got != "live" {
		t.Fatalf("LV busbar: %s", got)
	}
	described, _ := sections.Describe()
	if got := pyval.Get(described, "t1"); !pyval.Equal(got, []any{"live"}) {
		t.Fatalf("transformer states: %v", got)
	}
}

func TestEarthOnTheLVSideStaysOnTheLVSide(t *testing.T) {
	nodes, edges := transformerDiagram("dead")
	sections := NewSections(pyval.Obj("nodes", nodes, "edges", edges), map[string]any{"es": closedPosition})
	if got := sections.State("lv"); got != "earthed" {
		t.Fatalf("LV busbar: %s", got)
	}
	if got := sections.State("t1:a"); got != "dead" {
		t.Fatalf("an earthed LV side must not earth the HV side: %s", got)
	}
}

func TestDeadSourceLeavesBothSidesDead(t *testing.T) {
	nodes, edges := transformerDiagram("dead")
	sections := NewSections(pyval.Obj("nodes", nodes, "edges", edges), map[string]any{"es": openPosition})
	if sections.State("t1:a") != "dead" || sections.State("t1:b") != "dead" {
		t.Fatalf("got %s / %s", sections.State("t1:a"), sections.State("t1:b"))
	}
}
