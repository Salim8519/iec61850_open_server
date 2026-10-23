package project

import (
	"strings"
	"testing"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// IED components and GOOSE links (ADR-0112): contract bytes, plain-words refusals, one component
// per IED and IED removal. gooseIEDs gives testIEDs' ied0 one message G1 with a position (Ind4),
// two on/off values (Ind1, Ind5) and a number (Ind2, MX).

const g1 = "LD0/LLN0$GO$G1"

func gooseIEDs() []*pyjson.Object {
	ieds := testIEDs()
	members := []any{"GGIO1$ST$Ind4$stVal", "GGIO0$ST$Ind1$stVal", "GGIO1$ST$Ind5$stVal", "GGIO0$MX$Ind2$stVal"}
	refs := []any{"LD0/GGIO1.Ind4.stVal", "LD0/GGIO0.Ind1.stVal", "LD0/GGIO1.Ind5.stVal", "LD0/GGIO0.Ind2.stVal"}
	ieds[0].Set("datasets", []any{o("name", "DS", "members", members, "ref", "LD0/LLN0$DS", "fcda_refs", refs)}).
		Set("goose", o("publishes", []any{o("name", "G1", "gocb_ref", g1, "dataset", "DS", "dataset_ref", "LD0/LLN0$DS",
			"go_id", "G1", "app_id", int64(1), "conf_rev", int64(1), "members", members)}, "subscribes", []any{}))
	return ieds
}

func iedNode(id string, iedID any) *pyjson.Object {
	return o("id", id, "type", "equipment", "position", o("x", int64(0), "y", int64(0)),
		"data", o("label", "IED "+id, "kind", IEDComponentKind, "ied_id", iedID))
}

func link(mode string, refs ...any) *pyjson.Object {
	result := o("ied_id", "ied0", "gocb_ref", g1, "mode", mode)
	if mode == "contacts" {
		return result.Set("open", refs[0]).Set("closed", refs[1])
	}
	return result.Set("one", refs[0])
}

func TestIEDComponentBytesAndRefusals(t *testing.T) {
	ieds := gooseIEDs()
	fresh, err := ValidateDiagram(o("nodes", []any{o("id", "i", "type", "equipment", "position", o("x", int64(0), "y", int64(0)),
		"data", o("label", "IED-1", "kind", "IED"))}, "edges", []any{}), ieds)
	if err != nil {
		t.Fatal(err)
	}
	// Like a new point panel: every common default, no infeed fields, ied_id null at the end.
	want := `{"label": "IED-1", "kind": "IED", "rotation": 0, "state_mapping": null, "control_binding": null, "relay_statuses": [], "trip_command_polarity": "operate_false", "trip_state_polarity": "operated_true", "push_mode": "spc_momentary", "push_value": true, "push_rest_value": false, "push_reset_ms": 1000, "initial_position": "open", "active_color": null, "inactive_color": null, "ied_id": null}`
	if got := dumps(t, firstData(t, fresh)); got != want {
		t.Fatalf("a new IED component\n got: %s\nwant: %s", got, want)
	}
	cases := []struct {
		nodes []any
		want  string
	}{
		{[]any{iedNode("a", "ied0"), iedNode("b", "ied1"), iedNode("c", "ied0")}, "IED0 already has an IED component on this drawing"},
		{[]any{iedNode("a", "nope")}, "shows an IED that is not in the project"},
		{[]any{withData(iedNode("a", "ied0"), "state_mapping", direct("ied0", 1))}, "has no state mapping"},
		{[]any{withData(iedNode("a", "ied0"), "points", []any{point("status", "s", "ied0", 1)})}, "An IED component has no points"},
		{[]any{withData(iedNode("a", "ied0"), "goose_link", link("one", "LD0/GGIO1.Ind4.stVal"))}, "sends its messages from its own sheet"},
	}
	for _, item := range cases {
		_, err := ValidateDiagram(o("nodes", item.nodes, "edges", []any{}), ieds)
		if !strings.Contains(studioMessage(err), item.want) {
			t.Errorf("want %q, got %v", item.want, err)
		}
	}
	two, err := ValidateDiagram(o("nodes", []any{iedNode("a", "ied0"), iedNode("b", nil), iedNode("c", nil)}, "edges", []any{}), ieds)
	if err != nil || dumps(t, items(pyval.Lookup(two, "nodes", nil))[0]) == "" {
		t.Fatalf("components without an IED yet may repeat: %v", err)
	}
	_, err = ValidateDiagram(listDiagram("BOGUS", o()), ieds)
	if problem, ok := err.(*ContractError); !ok || strings.Contains(dumps(t, problem.Details), "'IED'") {
		t.Fatalf("the unknown-kind words stay those of the Python vectors: %v", err)
	}
}

func withData(node *pyjson.Object, key string, value any) *pyjson.Object {
	object(pyval.Lookup(node, "data", nil)).Set(key, value)
	return node
}

func TestGooseLinkBytesAndRefusals(t *testing.T) {
	ieds := gooseIEDs()
	result, err := ValidateDiagram(listDiagram("CB", o("goose_link", link("contacts", "LD0/GGIO0.Ind1.stVal", "LD0/GGIO1.Ind5.stVal"))), ieds)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dumps(t, pyval.Lookup(firstData(t, result), "goose_link", nil)),
		`{"ied_id": "ied0", "gocb_ref": "LD0/LLN0$GO$G1", "mode": "contacts", "one": null, "open": "LD0/GGIO0.Ind1.stVal", "closed": "LD0/GGIO1.Ind5.stVal"}`; got != want {
		t.Fatalf("link\n got: %s\nwant: %s", got, want)
	}
	cleared, err := ValidateDiagram(listDiagram("CB", o("goose_link", nil)), ieds)
	plain, _ := ValidateDiagram(listDiagram("CB", o()), ieds)
	if err != nil || dumps(t, cleared) != dumps(t, plain) || strings.Contains(dumps(t, plain), "goose_link") {
		t.Fatalf("a cleared link leaves the node's old bytes: %v", err)
	}
	cases := []struct {
		kind string
		link any
		want string
	}{
		{"JUNCTION", link("one", "LD0/GGIO1.Ind4.stVal"), "A junction is a drawing point, so it takes no GOOSE link"},
		{PointPanelKind, link("one", "LD0/GGIO1.Ind4.stVal"), "A point panel sends GOOSE through its own points"},
		{"CB", withLink("ied_id", "ied9"), "uses an IED that is not in the project"},
		{"CB", withLink("gocb_ref", "LD0/LLN0$GO$G9"), "IED0 does not send the GOOSE message"},
		{"CB", link("one", "LD0/GGIO1.Ind8.stVal"), "is not part of the GOOSE message G1"},
		{"CB", link("one", "LD0/GGIO0.Ind2.stVal"), "a switch sends a position"},
		{"CT", link("contacts", "LD0/GGIO0.Ind1.stVal", "LD0/GGIO1.Ind5.stVal"), "Only a breaker, disconnector or earth switch"},
	}
	for _, item := range cases {
		_, err := ValidateDiagram(listDiagram(item.kind, o("goose_link", item.link)), ieds)
		if !strings.Contains(studioMessage(err), item.want) {
			t.Errorf("%s: want %q, got %v", item.kind, item.want, err)
		}
	}
	_, err = ValidateDiagram(listDiagram("CB", o("goose_link", withLink("colour", "red"))), ieds)
	if problem, ok := err.(*ContractError); !ok || !strings.Contains(dumps(t, problem.Details), "extra_forbidden") {
		t.Fatalf("unknown link keys are refused: %v", err)
	}
	if _, err := ValidateDiagram(listDiagram("SOURCE", o("goose_link", link("one", "LD0/GGIO0.Ind2.stVal"))), ieds); err != nil {
		t.Fatalf("a number for a source: %v", err)
	}
}

func withLink(key string, value any) *pyjson.Object {
	return link("one", "LD0/GGIO1.Ind4.stVal").Set(key, value)
}

// Removing an IED removes its component (and any wire to it) and clears the GOOSE links that
// send its messages; the equipment stays.
func TestIEDRemovalTakesItsComponentAndLinks(t *testing.T) {
	cb := o("id", "cb", "type", "equipment", "position", o("x", int64(0), "y", int64(0)),
		"data", o("label", "CB", "kind", "CB", "goose_link", link("one", "LD0/GGIO1.Ind4.stVal")))
	diagram := o("nodes", []any{iedNode("comp0", "ied0"), iedNode("comp1", "ied1"), cb},
		"edges", []any{o("id", "w", "source", "comp0", "target", "cb")})
	project := o("ieds", anyList(gooseIEDs()), "diagram", diagram, "scenarios", []any{}, "sources", []any{}, "test_panel", []any{})
	ids := map[string]bool{"ied0": true}
	if got := dumps(t, Impacts(project, ids)); !strings.HasPrefix(got, `{"bindings": 2,`) {
		t.Fatalf("the component and the link count as bindings: %s", got)
	}
	after := WithoutIEDs(project, ids)
	nodes := items(pyval.Lookup(object(pyval.Lookup(after, "diagram", nil)), "nodes", nil))
	if len(nodes) != 2 || text(object(nodes[0]), "id") != "comp1" || text(object(nodes[1]), "id") != "cb" {
		t.Fatalf("nodes after removal: %s", dumps(t, nodes))
	}
	if _, present := object(pyval.Lookup(object(nodes[1]), "data", nil)).Get("goose_link"); present {
		t.Fatal("the breaker's link to the removed IED is cleared")
	}
	if edges := items(pyval.Lookup(object(pyval.Lookup(after, "diagram", nil)), "edges", nil)); len(edges) != 0 {
		t.Fatalf("a wire to the removed component goes with it: %s", dumps(t, edges))
	}
	if len(items(pyval.Lookup(object(pyval.Lookup(project, "diagram", nil)), "nodes", nil))) != 3 {
		t.Fatal("the project passed in is not changed")
	}
}

func anyList(ieds []*pyjson.Object) []any {
	result := make([]any, len(ieds))
	for index, ied := range ieds {
		result[index] = ied
	}
	return result
}
