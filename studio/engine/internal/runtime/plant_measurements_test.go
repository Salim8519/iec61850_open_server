package runtime

import (
	"reflect"
	"testing"

	"studio-engine/internal/runtime/pyval"
)

// Measurement list rows (ADR-0090) publish their raw value under "<node>:<row>", like relay rows.
func TestPlantStatesIncludeMeasurementRows(t *testing.T) {
	row := func(id, ref string) any {
		return pyval.Obj("id", id, "label", id, "source", pyval.Obj("ied_id", "ied1", "ref", ref, "report_ref", nil))
	}
	nodes := []any{
		pyval.Obj("id", "m1", "data", pyval.Obj("kind", "MEASUREMENT_LIST", "state_mapping", nil,
			"relay_statuses", []any{}, "measurements", []any{row("v", "LD0/MMXU1.PhV.mag.f"), row("i", "LD0/MMXU1.A.mag.f")})),
		pyval.Obj("id", "cb", "data", pyval.Obj("kind", "CB", "state_mapping", nil, "relay_statuses", []any{},
			"measurements", []any{row("x", "LD0/IGNORED.mag.f")})),
	}
	p := newPlantStates(&Coordinator{})
	p.reset(pyval.Obj("nodes", nodes, "edges", []any{}))
	if want := []string{"m1:v", "m1:i"}; !reflect.DeepEqual(p.order, want) {
		t.Fatalf("order = %v, want %v", p.order, want)
	}
	if got := p.bySource[bindingKey{"ied1", "LD0/MMXU1.A.mag.f"}]; !reflect.DeepEqual(got, []string{"m1:i"}) {
		t.Fatalf("a point change must republish its row, got %v", got)
	}
}
