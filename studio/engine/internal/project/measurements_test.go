package project

import (
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Measurement list (ADR-0090): contract, cross-checks and IED removal.

func row(id, ied string, k int, extra ...any) *pyjson.Object {
	entry := o("id", id, "label", "Row "+id, "source", o("ied_id", ied, "ref", signalRef(k), "report_ref", nil))
	for index := 0; index < len(extra); index += 2 {
		entry.Set(extra[index].(string), extra[index+1])
	}
	return entry
}

func listDiagram(kind string, data *pyjson.Object) *pyjson.Object {
	data.Set("label", "Feeder values").Set("kind", kind)
	node := o("id", "m1", "type", "equipment", "position", o("x", int64(0), "y", int64(0)), "data", data)
	return o("nodes", []any{node}, "edges", []any{})
}

func firstData(t *testing.T, diagram *pyjson.Object) *pyjson.Object {
	t.Helper()
	return object(pyval.Lookup(object(items(pyval.Lookup(diagram, "nodes", nil))[0]), "data", nil))
}

func studioMessage(err error) string {
	if e, ok := err.(*apperr.Error); ok {
		return e.Message
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestMeasurementDefaultsAndBytes(t *testing.T) {
	ieds := testIEDs()
	result, err := ValidateDiagram(listDiagram(MeasurementKind, o("measurements", []any{row("a", "ied0", 2)})), ieds)
	if err != nil {
		t.Fatal(err)
	}
	got := dumps(t, items(pyval.Lookup(firstData(t, result), "measurements", nil))[0])
	want := `{"id": "a", "label": "Row a", "source": {"ied_id": "ied0", "ref": "LD0/GGIO0.Ind2.stVal", ` +
		`"report_ref": null}, "scale": 1.0, "offset": 0.0, "unit": "", "decimals": 2}`
	if got != want {
		t.Fatalf("row dump\n got: %s\nwant: %s", got, want)
	}
	empty, err := ValidateDiagram(listDiagram(MeasurementKind, o()), ieds)
	if err != nil {
		t.Fatal(err)
	}
	if rows, present := firstData(t, empty).Get("measurements"); !present || len(items(rows)) != 0 {
		t.Fatalf("a new list must get an empty row list, got %v", rows)
	}
	breaker, err := ValidateDiagram(listDiagram("CB", o("measurements", []any{})), ieds)
	if err != nil {
		t.Fatal(err)
	}
	if _, present := firstData(t, breaker).Get("measurements"); present {
		t.Fatal("other kinds must not carry a measurements key (their saved bytes stay unchanged)")
	}
	plain, _ := ValidateDiagram(listDiagram("CB", o()), ieds)
	if strings.Contains(dumps(t, plain), "measurements") {
		t.Fatal("a diagram without measurement lists must dump as before")
	}
}

func TestMeasurementContractErrors(t *testing.T) {
	ieds := testIEDs()
	many := []any{}
	for n := 0; n < 51; n++ {
		many = append(many, row(strings.Repeat("r", 1)+string(rune('A'+n%26))+string(rune('a'+n/26)), "ied0", 2))
	}
	cases := []struct {
		rows []any
		want string
	}{
		{[]any{row("a", "ied0", 2, "scale", "abc")}, "float_parsing"},
		{[]any{row("a", "ied0", 2, "scale", 2e9)}, "less_than_equal"},
		{[]any{row("a", "ied0", 2, "decimals", int64(7))}, "less_than_equal"},
		{[]any{row("a", "ied0", 2, "unit", strings.Repeat("u", 17))}, "string_too_long"},
		{[]any{row("a", "ied0", 2, "colour", "red")}, "extra_forbidden"},
		{[]any{row("bad id", "ied0", 2)}, "string_pattern_mismatch"},
		{many, "too_long"},
	}
	for _, item := range cases {
		_, err := ValidateDiagram(listDiagram(MeasurementKind, o("measurements", item.rows)), ieds)
		problem, ok := err.(*ContractError)
		if !ok || !strings.Contains(dumps(t, problem.Details), item.want) {
			t.Errorf("want %s, got %v", item.want, err)
		}
	}
}

func TestMeasurementCrossChecks(t *testing.T) {
	ieds := testIEDs()
	cases := []struct {
		kind string
		data *pyjson.Object
		want string
	}{
		{MeasurementKind, o("measurements", []any{row("a", "ied0", 1)}), `needs a number signal (integer or float), but LD0/GGIO0.Ind1.stVal is BOOLEAN`},
		{MeasurementKind, o("measurements", []any{row("a", "ied0", 2), row("a", "ied0", 3)}), "Measurement identifiers must be unique"},
		{MeasurementKind, o("measurements", []any{row("a", "nope", 2)}), "A signal binding references an unknown IED"},
		{MeasurementKind, o("measurements", []any{row("a", "ied0", 9999)}), "A signal binding references an unknown signal"},
		{"CB", o("measurements", []any{row("a", "ied0", 2)}), "Only a measurement list accepts measurements"},
		{MeasurementKind, o("state_mapping", direct("ied0", 2)), "A measurement list uses its own measurement rows"},
		{MeasurementKind, o("control_binding", o("ied_id", "ied0", "ref", "LD0/CSWI1.Pos")), "does not accept a control binding"},
	}
	for _, item := range cases {
		_, err := ValidateDiagram(listDiagram(item.kind, item.data), ieds)
		if !strings.Contains(studioMessage(err), item.want) {
			t.Errorf("want %q, got %q", item.want, studioMessage(err))
		}
	}
	good := o("measurements", []any{row("v", "ied0", 2, "scale", 0.001, "unit", "kV", "decimals", int64(3)),
		row("i", "ied1", 3, "offset", int64(-4), "unit", "A")})
	if _, err := ValidateDiagram(listDiagram(MeasurementKind, good), ieds); err != nil {
		t.Fatalf("FLOAT32 MX and INT32 CF rows are numbers: %v", err)
	}
}

func TestMeasurementNodeData(t *testing.T) {
	ieds := testIEDs()
	lookup := LinearLookup(ieds)
	current, err := ValidateDiagramWith(listDiagram(MeasurementKind, o()), lookup)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := NodeDataDiagram(current, "m1", o("measurements", []any{row("a", "ied2", 6, "scale", int64(1000))}), lookup, true)
	if err != nil {
		t.Fatal(err)
	}
	entry := object(items(pyval.Lookup(firstData(t, changed), "measurements", nil))[0])
	if scale, _ := entry.Get("scale"); scale != 1000.0 {
		t.Fatalf("an integer scale is stored as a float, got %#v", scale)
	}
	if _, err := NodeDataDiagram(current, "m1", o("measurements", []any{row("a", "ied2", 5)}), lookup, true); err == nil {
		t.Fatal("a Boolean row must be refused through node data too")
	}
}

func TestMeasurementRemoval(t *testing.T) {
	ieds := testIEDs()
	diagram, err := ValidateDiagram(listDiagram(MeasurementKind, o("measurements", []any{
		row("a", "ied0", 2), row("b", "ied1", 2)})), ieds)
	if err != nil {
		t.Fatal(err)
	}
	project := o("ieds", []any{ieds[0], ieds[1]}, "diagram", diagram, "sources", []any{}, "test_panel", []any{}, "scenarios", []any{})
	gone := map[string]bool{"ied0": true}
	if bindings, _ := Impacts(project, gone).Get("bindings"); bindings != int64(1) {
		t.Fatalf("the list reads from ied0, impacts bindings = %v", bindings)
	}
	rows := items(pyval.Lookup(firstData(t, object(pyval.Lookup(WithoutIEDs(project, gone), "diagram", nil))), "measurements", nil))
	if len(rows) != 1 || text(object(rows[0]), "id") != "b" {
		t.Fatalf("only the ied1 row stays, got %v", rows)
	}
	if left := items(pyval.Lookup(firstData(t, diagram), "measurements", nil)); len(left) != 2 {
		t.Fatal("WithoutIEDs must not change the original project")
	}
}
