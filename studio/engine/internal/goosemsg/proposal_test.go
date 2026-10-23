package goosemsg

import (
	"reflect"
	"testing"

	"studio-engine/internal/pyjson"
)

func node(id, kind string, x, y any, iedID any) any {
	return o("id", id, "type", "equipment", "position", o("x", x, "y", y),
		"data", o("label", id, "kind", kind, "rotation", int64(0), "ied_id", iedID))
}

func ied(id string, publishes, subscribes int) *pyjson.Object {
	list := func(count int) []any {
		items := []any{}
		for index := 0; index < count; index++ {
			items = append(items, o("gocb_ref", id+"/LLN0$GO$G"))
		}
		return items
	}
	return o("id", id, "name", "N_"+id, "goose", o("publishes", list(publishes), "subscribes", list(subscribes)))
}

// Every project IED that sends or receives GOOSE and has no component yet, in project order, in
// one row 60 px above the drawing's top (card 120 high), spread over the drawing's width; once
// components exist, their row continues to the right.
func TestProposeOneRowAboveTheDrawing(t *testing.T) {
	nodes := []any{node("cb", "CB", int64(100), int64(300), nil), node("ds", "DS", 1100.0, int64(500), nil)}
	ieds := []*pyjson.Object{ied("a", 1, 0), ied("b", 1, 1), ied("c", 0, 0), ied("d", 0, 2), ied("e", 2, 0)}
	got := Propose(nodes, ieds)
	// top = 300 -> row y = 300 - 60 - 120 = 120; 4 cards from x 100 to 1100: pitch snap(1000 / 3) = 340
	want := []Placement{{"a", "N_a", 100, 120}, {"b", "N_b", 440, 120}, {"d", "N_d", 780, 120}, {"e", "N_e", 1120, 120}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("proposal %+v, want %+v", got, want)
	}
	narrow := Propose([]any{node("cb", "CB", int64(0), int64(0), nil)}, ieds[:2])
	if want := []Placement{{"a", "N_a", 0, -180}, {"b", "N_b", 240, -180}}; !reflect.DeepEqual(narrow, want) {
		t.Fatalf("a narrow drawing keeps the card pitch: %+v", narrow)
	}
	withRow := append(nodes, node("shown", IEDKind, int64(400), int64(100), "b"), node("empty", IEDKind, int64(160), int64(100), nil))
	if got, want := Propose(withRow, ieds), []Placement{{"a", "N_a", 640, 100}, {"d", "N_d", 880, 100}, {"e", "N_e", 1120, 100}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("the components' row continues to the right: %+v", got)
	}
	if got := Propose(nil, []*pyjson.Object{ied("a", 1, 0)}); !reflect.DeepEqual(got, []Placement{{"a", "N_a", 0, 0}}) {
		t.Fatalf("empty drawing: %+v", got)
	}
	if Propose(withRow, []*pyjson.Object{ied("b", 1, 0), ied("c", 0, 0)}) != nil {
		t.Fatal("nothing to add gives nothing")
	}
}
