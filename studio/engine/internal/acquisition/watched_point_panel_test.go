package acquisition

import (
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// A real IED polls the signals of point panel points (ADR-0102) with their quality, and nothing
// for command points, which have a control and no signal.
func TestWatchedPointsIncludePointPanels(t *testing.T) {
	signal := func(ref, fc, quality string) *pyjson.Object {
		item := pyv.O("ref", ref, "fc", fc)
		if quality != "" {
			item.Set("quality_ref", quality)
		}
		return item
	}
	ied := pyv.O("id", "ied1", "signals", []any{
		signal("LD0/XCBR1.Pos.stVal", "ST", "LD0/XCBR1.Pos.q"),
		signal("LD0/XCBR1.Pos.q", "ST", ""),
		signal("LD0/GGIO1.Ind1.stVal", "ST", ""),
		signal("LD0/MMXU1.A.mag.f", "MX", ""),
	})
	status := func(id, iedID, kind, ref string) *pyjson.Object {
		return pyv.O("id", id, "label", id, "type", kind, "source", pyv.O("ied_id", iedID, "ref", ref, "report_ref", nil),
			"control", nil)
	}
	command := pyv.O("id", "c", "label", "c", "type", "command", "source", nil,
		"control", pyv.O("ied_id", "ied1", "ref", "LD0/XCBR1.Pos"))
	panel := pyv.O("id", "pp", "data", pyv.O("kind", "POINT_PANEL", "state_mapping", nil, "ied_id", "ied1", "points", []any{
		status("d", "ied1", "double", "LD0/XCBR1.Pos.stVal"), command, status("s", "ied1", "status", "LD0/GGIO1.Ind1.stVal"),
		status("d2", "ied1", "double", "LD0/XCBR1.Pos.stVal")}))
	project := pyv.O("diagram", pyv.O("nodes", []any{panel}), "test_panel", []any{})
	got := WatchedPoints(project, ied)
	want := []string{"LD0/XCBR1.Pos.stVal", "LD0/XCBR1.Pos.q", "LD0/GGIO1.Ind1.stVal"}
	if len(got) != len(want) {
		t.Fatalf("got %d points, want %v", len(got), want)
	}
	for index, ref := range want {
		if pyv.GetStr(got[index].(*pyjson.Object), "ref") != ref {
			t.Fatalf("point %d = %v, want %s", index, got[index], ref)
		}
	}
}
