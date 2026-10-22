package project

import (
	"fmt"
	"strings"
	"testing"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Point panel (ADR-0102): contract, cross-checks, node data and IED removal. testIEDs signal k
// is Dbpos (k%4 = 0), BOOLEAN (1), FLOAT32 MX (2) or INT32 CF (3); control n is DPC (even) or SPC (odd).

func pointSignal(ied string, k int) *pyjson.Object {
	return o("ied_id", ied, "ref", signalRef(k), "report_ref", nil)
}

// point builds a status, double or value row reading signal k of ied.
func point(kind, id, ied string, k int, extra ...any) *pyjson.Object {
	entry := o("id", id, "label", "Point "+id, "type", kind, "source", pointSignal(ied, k))
	for index := 0; index < len(extra); index += 2 {
		entry.Set(extra[index].(string), extra[index+1])
	}
	return entry
}

func command(id, ied string, n int, extra ...any) *pyjson.Object {
	entry := o("id", id, "label", "Command "+id, "type", "command",
		"control", o("ied_id", ied, "ref", fmt.Sprintf("LD0/CSWI%d.Pos", n)))
	for index := 0; index < len(extra); index += 2 {
		entry.Set(extra[index].(string), extra[index+1])
	}
	return entry
}

func panelData(ied any, rows ...any) *pyjson.Object {
	return o("ied_id", ied, "points", rows)
}

func validatePanel(data *pyjson.Object) (*pyjson.Object, error) {
	return ValidateDiagram(listDiagram(PointPanelKind, data), testIEDs())
}

func TestPointPanelDefaultsAndBytes(t *testing.T) {
	result, err := validatePanel(panelData("ied0", point("status", "s", "ied0", 1), point("double", "d", "ied0", 0),
		command("c", "ied0", 1), point("value", "v", "ied0", 2, "unit", "V", "scale", 0.001, "pinned", true, "healthy", 230)))
	if err != nil {
		t.Fatal(err)
	}
	rows := items(pyval.Lookup(firstData(t, result), "points", nil))
	want := []string{
		`{"id": "s", "label": "Point s", "group": "", "type": "status", "source": {"ied_id": "ied0", "ref": "LD0/GGIO0.Ind1.stVal", "report_ref": null}, "control": null, "unit": "", "scale": 1.0, "offset": 0.0, "decimals": 2, "pinned": false, "healthy": null}`,
		`{"id": "d", "label": "Point d", "group": "", "type": "double", "source": {"ied_id": "ied0", "ref": "LD0/GGIO0.Ind0.stVal", "report_ref": null}, "control": null, "unit": "", "scale": 1.0, "offset": 0.0, "decimals": 2, "pinned": false, "healthy": null}`,
		`{"id": "c", "label": "Command c", "group": "", "type": "command", "source": null, "control": {"ied_id": "ied0", "ref": "LD0/CSWI1.Pos"}, "unit": "", "scale": 1.0, "offset": 0.0, "decimals": 2, "pinned": false, "healthy": null}`,
		`{"id": "v", "label": "Point v", "group": "", "type": "value", "source": {"ied_id": "ied0", "ref": "LD0/GGIO0.Ind2.stVal", "report_ref": null}, "control": null, "unit": "V", "scale": 0.001, "offset": 0.0, "decimals": 2, "pinned": true, "healthy": 230}`,
	}
	for index, expected := range want {
		if got := dumps(t, rows[index]); got != expected {
			t.Errorf("row %d\n got: %s\nwant: %s", index, got, expected)
		}
	}
	empty, err := validatePanel(o())
	if err != nil {
		t.Fatal(err)
	}
	data := firstData(t, empty)
	if list, present := data.Get("points"); !present || len(items(list)) != 0 {
		t.Fatalf("a new panel must get an empty point list, got %v", list)
	}
	if iedID, present := data.Get("ied_id"); !present || iedID != nil {
		t.Fatalf("a new panel has no IED yet, got %v", iedID)
	}
	breaker, err := ValidateDiagram(listDiagram("CB", o("points", []any{}, "ied_id", nil)), testIEDs())
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"points", "ied_id"} {
		if _, present := firstData(t, breaker).Get(key); present {
			t.Fatalf("other kinds must not carry %s (their saved bytes stay unchanged)", key)
		}
	}
	plain, _ := ValidateDiagram(listDiagram("CB", o()), testIEDs())
	if text := dumps(t, plain); strings.Contains(text, "points") || strings.Contains(text, "ied_id") {
		t.Fatal("a diagram without point panels must dump as before")
	}
}

func TestPointPanelContractErrors(t *testing.T) {
	many := []any{}
	for n := 0; n < 301; n++ {
		many = append(many, point("status", fmt.Sprintf("p%d", n), "ied0", 1))
	}
	cases := []struct {
		rows []any
		want string
	}{
		{many, "too_long"},
		{[]any{point("status", "a", "ied0", 1, "colour", "red")}, "extra_forbidden"},
		{[]any{point("status", "bad id", "ied0", 1)}, "string_pattern_mismatch"},
		{[]any{point("status", "a", "ied0", 1, "label", "")}, "string_too_short"},
		{[]any{point("status", "a", "ied0", 1, "group", strings.Repeat("g", 101))}, "string_too_long"},
		{[]any{point("switch", "a", "ied0", 1)}, "literal_error"},
		{[]any{point("status", "a", "ied0", 1, "healthy", "yes")}, "healthy_type"},
		{[]any{point("value", "a", "ied0", 2, "scale", 2e9)}, "less_than_equal"},
		{[]any{point("value", "a", "ied0", 2, "decimals", int64(7))}, "less_than_equal"},
	}
	for _, item := range cases {
		_, err := validatePanel(panelData("ied0", item.rows...))
		problem, ok := err.(*ContractError)
		if !ok || !strings.Contains(dumps(t, problem.Details), item.want) {
			t.Errorf("want %s, got %v", item.want, err)
		}
	}
	exact := make([]any, 300)
	for n := range exact {
		exact[n] = point("status", fmt.Sprintf("p%d", n), "ied0", 1)
	}
	if _, err := validatePanel(panelData("ied0", exact...)); err != nil {
		t.Fatalf("300 points are allowed: %v", err)
	}
}

func TestPointPanelCrossChecks(t *testing.T) {
	cases := []struct {
		kind string
		data *pyjson.Object
		want string
	}{
		{PointPanelKind, panelData("nope"), "unknown IED"},
		{PointPanelKind, panelData(nil, point("status", "a", "ied0", 1)), "Choose the IED of this point panel"},
		{PointPanelKind, panelData("ied0", point("status", "a", "ied0", 1), point("double", "a", "ied0", 0)), "must be unique"},
		{PointPanelKind, panelData("ied0", point("status", "a", "ied1", 1)), "reads from another IED"},
		{PointPanelKind, panelData("ied0", command("a", "ied1", 1)), "sends to another IED"},
		{PointPanelKind, panelData("ied0", point("status", "a", "ied0", 0)), "needs an on/off (Boolean) signal, but LD0/GGIO0.Ind0.stVal is Dbpos"},
		{PointPanelKind, panelData("ied0", point("double", "a", "ied0", 1)), "needs a position (Dbpos) signal"},
		{PointPanelKind, panelData("ied0", point("value", "a", "ied0", 3)), "needs a measured number (functional constraint MX)"},
		{PointPanelKind, panelData("ied0", point("value", "a", "ied0", 1)), "needs a measured number"},
		{PointPanelKind, panelData("ied0", point("status", "a", "ied0", 9999)), "uses a signal this IED does not have"},
		{PointPanelKind, panelData("ied0", command("a", "ied0", 99)), "uses a control this IED does not have"},
		{PointPanelKind, panelData("ied0", o("id", "a", "label", "A", "type", "status")), "needs a signal"},
		{PointPanelKind, panelData("ied0", o("id", "a", "label", "A", "type", "command")), "needs a control"},
		{PointPanelKind, panelData("ied0", point("command", "a", "ied0", 1)), "is a command, so it uses a control and no signal"},
		{PointPanelKind, panelData("ied0", command("a", "ied0", 1, "source", pointSignal("ied0", 1))), "uses a control and no signal"},
		{PointPanelKind, panelData("ied0", point("status", "a", "ied0", 1, "control", o("ied_id", "ied0", "ref", "LD0/CSWI1.Pos"))), "uses a signal and no control"},
		{PointPanelKind, panelData("ied0", point("status", "a", "ied0", 1, "unit", "V")), "not a value point"},
		{PointPanelKind, panelData("ied0", point("double", "a", "ied0", 0, "scale", 2.0)), "not a value point"},
		{PointPanelKind, panelData("ied0", point("status", "a", "ied0", 1, "pinned", true)), "cannot be pinned"},
		{PointPanelKind, panelData("ied0", point("value", "a", "ied0", 2, "scale", 0)), "scale 0"},
		{PointPanelKind, panelData("ied0", point("status", "a", "ied0", 1, "healthy", 1)), "healthy value is on (true) or off (false)"},
		{PointPanelKind, panelData("ied0", point("double", "a", "ied0", 0, "healthy", 4)), "healthy value is 0 (intermediate)"},
		{PointPanelKind, panelData("ied0", point("double", "a", "ied0", 0, "healthy", true)), "healthy value is 0 (intermediate)"},
		{PointPanelKind, panelData("ied0", point("value", "a", "ied0", 2, "healthy", true)), "healthy value is a number"},
		{PointPanelKind, panelData("ied0", command("a", "ied0", 1, "healthy", true)), "cannot have a healthy value"},
		{PointPanelKind, o("state_mapping", direct("ied0", 1)), "uses its own points instead of a single state mapping"},
		{PointPanelKind, o("control_binding", o("ied_id", "ied0", "ref", "LD0/CSWI1.Pos")), "does not accept a control binding"},
		{"CB", o("points", []any{point("status", "a", "ied0", 1)}), "Only a point panel accepts an IED and points"},
		{"CB", o("ied_id", "ied0"), "Only a point panel accepts an IED and points"},
		{MeasurementKind, o("points", []any{point("status", "a", "ied0", 1)}), "Only a point panel accepts"},
	}
	for _, item := range cases {
		_, err := ValidateDiagram(listDiagram(item.kind, item.data), testIEDs())
		if !strings.Contains(studioMessage(err), item.want) {
			t.Errorf("want %q, got %q", item.want, studioMessage(err))
		}
	}
	good := panelData("ied1", point("status", "s", "ied1", 5, "healthy", true, "group", "Bay"),
		point("double", "d", "ied1", 4, "healthy", int64(2)), point("value", "v", "ied1", 6, "offset", int64(-4), "healthy", 49.9),
		command("spc", "ied1", 1), command("dpc", "ied1", 2))
	if _, err := validatePanel(good); err != nil {
		t.Fatalf("a valid panel of every type: %v", err)
	}
}

func TestPointPanelNodeData(t *testing.T) {
	lookup := LinearLookup(testIEDs())
	current, err := ValidateDiagramWith(listDiagram(PointPanelKind, o()), lookup)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := NodeDataDiagram(current, "m1", panelData("ied2", point("status", "a", "ied2", 5)), lookup, true)
	if err != nil {
		t.Fatal(err)
	}
	if rows := items(pyval.Lookup(firstData(t, changed), "points", nil)); len(rows) != 1 {
		t.Fatalf("the point is stored, got %v", rows)
	}
	if _, err := NodeDataDiagram(current, "m1", panelData("ied2", point("status", "a", "ied2", 6)), lookup, true); err == nil {
		t.Fatal("a Boolean status point must not accept a float signal, through node data too")
	}
}

func TestPointPanelRemoval(t *testing.T) {
	ieds := testIEDs()
	diagram, err := ValidateDiagram(listDiagram(PointPanelKind, panelData("ied0", point("status", "a", "ied0", 1),
		command("c", "ied0", 1))), ieds)
	if err != nil {
		t.Fatal(err)
	}
	project := o("ieds", []any{ieds[0], ieds[1]}, "diagram", diagram, "sources", []any{}, "test_panel", []any{}, "scenarios", []any{})
	if bindings, _ := Impacts(project, map[string]bool{"ied0": true}).Get("bindings"); bindings != int64(1) {
		t.Fatalf("the panel belongs to ied0, impacts bindings = %v", bindings)
	}
	if bindings, _ := Impacts(project, map[string]bool{"ied1": true}).Get("bindings"); bindings != int64(0) {
		t.Fatalf("removing another IED leaves the panel alone, got %v", bindings)
	}
	data := firstData(t, object(pyval.Lookup(WithoutIEDs(project, map[string]bool{"ied0": true}), "diagram", nil)))
	if iedID, _ := data.Get("ied_id"); iedID != nil {
		t.Fatalf("the panel stays without an IED, got %v", iedID)
	}
	if rows := items(pyval.Lookup(data, "points", nil)); len(rows) != 0 {
		t.Fatalf("the points of the removed IED go, got %v", rows)
	}
	if left := items(pyval.Lookup(firstData(t, diagram), "points", nil)); len(left) != 2 {
		t.Fatal("WithoutIEDs must not change the original project")
	}
	kept := firstData(t, object(pyval.Lookup(WithoutIEDs(project, map[string]bool{"ied1": true}), "diagram", nil)))
	if len(items(pyval.Lookup(kept, "points", nil))) != 2 {
		t.Fatal("removing another IED keeps the points")
	}
}

// The text of the unknown-kind error is the one the frozen Python vectors record plus the
// Go-only MEASUREMENT_LIST: POINT_PANEL is accepted without being named there (ADR-0102).
func TestUnknownKindMessageIsUnchanged(t *testing.T) {
	_, err := ValidateDiagram(listDiagram("BUSBAR", o()), testIEDs())
	problem, ok := err.(*ContractError)
	if !ok || !strings.Contains(dumps(t, problem.Details), "'JUNCTION', 'INFEED' or 'MEASUREMENT_LIST'") ||
		strings.Contains(dumps(t, problem.Details), "POINT_PANEL") {
		t.Fatalf("unknown kind message changed: %v", err)
	}
}
