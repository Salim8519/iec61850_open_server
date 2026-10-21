package acquisition

import (
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// A real IED polls the points of measurement list rows (ADR-0090) with their quality.
func TestWatchedPointsIncludeMeasurements(t *testing.T) {
	signal := func(ref, fc, quality string) *pyjson.Object {
		item := pyv.O("ref", ref, "fc", fc)
		if quality != "" {
			item.Set("quality_ref", quality)
		}
		return item
	}
	ied := pyv.O("id", "ied1", "signals", []any{
		signal("LD0/MMXU1.PhV.phsA.cVal.mag.f", "MX", "LD0/MMXU1.PhV.phsA.q"),
		signal("LD0/MMXU1.PhV.phsA.q", "MX", ""),
		signal("LD0/MMXU1.A.phsA.cVal.mag.f", "MX", ""),
		signal("LD0/GGIO1.Ind1.stVal", "ST", ""),
	})
	row := func(id, iedID, ref string) *pyjson.Object {
		return pyv.O("id", id, "label", id, "source", pyv.O("ied_id", iedID, "ref", ref, "report_ref", nil))
	}
	list := pyv.O("id", "m1", "data", pyv.O("kind", "MEASUREMENT_LIST", "state_mapping", nil, "measurements", []any{
		row("v", "ied1", "LD0/MMXU1.PhV.phsA.cVal.mag.f"), row("other", "ied2", "LD0/MMXU1.A.phsA.cVal.mag.f"),
		row("i", "ied1", "LD0/MMXU1.A.phsA.cVal.mag.f")}))
	project := pyv.O("diagram", pyv.O("nodes", []any{list}), "test_panel", []any{})
	got := WatchedPoints(project, ied)
	want := []string{"LD0/MMXU1.PhV.phsA.cVal.mag.f", "LD0/MMXU1.PhV.phsA.q", "LD0/MMXU1.A.phsA.cVal.mag.f"}
	if len(got) != len(want) {
		t.Fatalf("got %d points, want %v", len(got), want)
	}
	for index, ref := range want {
		if pyv.GetStr(got[index].(*pyjson.Object), "ref") != ref {
			t.Fatalf("point %d = %v, want %s", index, got[index], ref)
		}
	}
}
