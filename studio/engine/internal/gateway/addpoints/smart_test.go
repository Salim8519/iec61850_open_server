package addpoints

import (
	"testing"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

func byID(points []any) map[string]*pyjson.Object {
	result := map[string]*pyjson.Object{}
	for _, value := range points {
		point := value.(*pyjson.Object)
		result[text(point, "id")] = point
	}
	return result
}

// TestSmartReplaceNamesAndPairs: smart Propose keeps Propose's selection, names points from
// the SCL, adds the status of every command and pairs the command's address with it.
func TestSmartReplaceNamesAndPairs(t *testing.T) {
	ieds := project()
	plain, _, _ := gateway.Propose(ieds, "iec104")
	points, _, summary, err := Smart(ieds, "iec104", SmartOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found := byID(points)
	breaker, command := found["f1:FEED1LD0/XCBR1.Pos.stVal:double"], found["f1:FEED1LD0/XCBR1.Pos:command_double"]
	spare := found["f1:FEED1LD0/GGIO1.SPCSO1:command_single"]
	spareStatus := found["f1:FEED1LD0/GGIO1.SPCSO1.stVal:single"]
	if breaker == nil || command == nil || spare == nil || spareStatus == nil {
		t.Fatalf("points %s", pointJSON(t, points))
	}
	if text(breaker, "description") != "Breaker position" || text(spare, "description") != "Spare output" {
		t.Fatalf("names %s", pointJSON(t, points))
	}
	offset := func(point *pyjson.Object) int64 {
		at, _ := point.Get("address")
		return at.(int64) - gateway.BlockStart("iec104", text(point, "kind"))
	}
	if offset(command) != offset(breaker) || offset(spare) != offset(spareStatus) {
		t.Fatalf("pairs %s", pointJSON(t, points))
	}
	if len(points) != len(plain) { // every status was proposed already: nothing to add
		t.Fatalf("count %d vs %d: %s", len(points), len(plain), pointJSON(t, points))
	}
	if _, problems, err := gateway.ValidatePoints("iec104", points, ieds); err != nil || len(problems) != 0 {
		t.Fatalf("validate %v %v", problems, err)
	}
	if len(summary) != 2 || summary[0].Name != "FEED1" || summary[0].Offered == 0 || summary[1].Role != "real" ||
		!summary[1].Ticked {
		t.Fatalf("summary %+v", summary)
	}
}

// TestSmartAddOnlyNew: the list keeps every point exactly as the user edited it; only
// missing points are added, at free addresses; unticked IEDs add nothing.
func TestSmartAddOnlyNew(t *testing.T) {
	ieds := project()
	edited := pyjson.NewObject().Set("id", "f1:FEED1LD0/XCBR1.Pos.stVal:double").Set("kind", "double").Set("ied", "f1").
		Set("ref", "FEED1LD0/XCBR1.Pos.stVal").Set("address", int64(20050)).Set("scale", 1.0).Set("description", "My CB")
	own := pyjson.NewObject().Set("id", "own:1").Set("kind", "double").Set("source", "manual").Set("address", int64(20001)).
		Set("scale", 1.0).Set("description", "Spare")
	current := []any{edited, own}
	points, notes, summary, err := Smart(ieds, "iec104", SmartOptions{AddNew: true, Current: current,
		Ticked: map[string]bool{"f1": true}})
	if err != nil {
		t.Fatal(err)
	}
	found := byID(points)
	if points[0] != edited || points[1] != own || text(found[text(edited, "id")], "description") != "My CB" {
		t.Fatalf("edits lost %s", pointJSON(t, points))
	}
	for _, value := range points[2:] {
		point := value.(*pyjson.Object)
		if text(point, "ied") != "f1" || text(point, "id") == text(edited, "id") {
			t.Fatalf("added %s", pointJSON(t, point))
		}
	}
	command := found["f1:FEED1LD0/XCBR1.Pos:command_double"]
	if at, _ := command.Get("address"); at != int64(50050) { // paired with the edited breaker at 20050
		t.Fatalf("command %s", pointJSON(t, command))
	}
	if _, problems, err := gateway.ValidatePoints("iec104", points, ieds); err != nil || len(problems) != 0 {
		t.Fatalf("validate %v %v", problems, err)
	}
	if len(notes) != 1 || summary[0].New != summary[0].Offered-1 || summary[1].Ticked {
		t.Fatalf("notes %v summary %+v", notes, summary)
	}
	again, notes, _, _ := Smart(ieds, "iec104", SmartOptions{AddNew: true, Current: points, Ticked: map[string]bool{"f1": true}})
	if len(again) != len(points) || notes[0] != "No new points: every proposed point is already in the list. Your list is unchanged." {
		t.Fatalf("second run %d %v", len(again), notes)
	}
}
