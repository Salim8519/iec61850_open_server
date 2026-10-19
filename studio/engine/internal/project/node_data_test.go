package project

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func o(pairs ...any) *pyjson.Object {
	result := pyjson.NewObject()
	for index := 0; index < len(pairs); index += 2 {
		result.Set(pairs[index].(string), pairs[index+1])
	}
	return result
}

// testIEDs builds IEDs with every signal family the diagram checks look at.
func testIEDs() []*pyjson.Object {
	var ieds []*pyjson.Object
	for n := 0; n < 3; n++ {
		signals, controls := []any{}, []any{}
		for k := 0; k < 120; k++ {
			kind, fc, writable := []string{"Dbpos", "BOOLEAN", "FLOAT32", "INT32"}[k%4], []string{"ST", "ST", "MX", "CF"}[k%4], k%7 != 0
			signals = append(signals, o("ref", fmt.Sprintf("LD0/GGIO%d.Ind%d.stVal", k/4, k), "kind", kind, "fc", fc, "writable", writable))
		}
		for k := 0; k < 40; k++ {
			controls = append(controls, o("ref", fmt.Sprintf("LD0/CSWI%d.Pos", k), "cdc", []string{"DPC", "SPC"}[k%2]))
		}
		ieds = append(ieds, o("id", fmt.Sprintf("ied%d", n), "name", fmt.Sprintf("IED%d", n), "signals", signals, "controls", controls))
	}
	return ieds
}

func signalRef(k int) string { return fmt.Sprintf("LD0/GGIO%d.Ind%d.stVal", k/4, k) }

func direct(ied string, k int) *pyjson.Object {
	return o("mode", "direct", "source", o("ied_id", ied, "ref", signalRef(k), "report_ref", nil))
}

// testDiagram has one node of every kind with a valid binding where the kind takes one.
func testDiagram() *pyjson.Object {
	nodes, edges := []any{}, []any{}
	add := func(kind string, data *pyjson.Object) {
		id := fmt.Sprintf("n%d", len(nodes))
		data.Set("label", kind+" "+id).Set("kind", kind)
		nodes = append(nodes, o("id", id, "type", "equipment", "position", o("x", int64(len(nodes)*20), "y", 5.5), "data", data))
		if len(nodes) > 1 {
			edges = append(edges, o("id", "w"+id, "source", fmt.Sprintf("n%d", len(nodes)-2), "target", id))
		}
	}
	for n := 0; n < 3; n++ {
		ied := fmt.Sprintf("ied%d", n)
		add("CB", o("state_mapping", direct(ied, 8+4*n), "control_binding", o("ied_id", ied, "ref", "LD0/CSWI2.Pos")))
		add("DS", o("state_mapping", direct(ied, 12)))
		add("ES", o())
		add("TRIP_RELAY", o("state_mapping", direct(ied, 1), "control_binding", o("ied_id", ied, "ref", "LD0/CSWI1.Pos")))
		add("PUSH_BUTTON", o("push_mode", "latching_status", "state_mapping", direct(ied, 5)))
		add("PUSH_BUTTON", o("control_binding", o("ied_id", ied, "ref", "LD0/CSWI3.Pos")))
		add("SOURCE", o("state_mapping", direct(ied, 2)))
		add("RELAY", o("relay_statuses", []any{o("id", "trip", "label", "Trip", "source", o("ied_id", ied, "ref", signalRef(9)))}))
		add("JUNCTION", o())
		add("INFEED", o("infeed_type", "grid"))
		add("CT", o("state_mapping", direct(ied, 6)))
	}
	return o("nodes", nodes, "edges", edges)
}

var testChanges = []func(ied string) *pyjson.Object{
	func(string) *pyjson.Object { return o("initial_position", "closed") },
	func(string) *pyjson.Object { return o("initial_position", "open") },
	func(string) *pyjson.Object { return o("initial_position", "bogus") },
	func(string) *pyjson.Object { return o("label", "") },
	func(string) *pyjson.Object { return o("label", strings.Repeat("x", 101)) },
	func(string) *pyjson.Object { return o("kind", "NOPE", "rotation", "x") },
	func(string) *pyjson.Object { return o("rotation", "90") },
	func(string) *pyjson.Object { return o("state_mapping", nil) },
	func(string) *pyjson.Object { return o("control_binding", nil) },
	func(string) *pyjson.Object { return o("extra", int64(1)) },
	func(string) *pyjson.Object { return o("infeed_type", "line") },
	func(string) *pyjson.Object { return o("infeed_state", "dead", "infeed_type", nil) },
	func(string) *pyjson.Object { return o("push_mode", "self_resetting") },
	func(string) *pyjson.Object { return o("active_color", "#12zz56") },
	func(string) *pyjson.Object { return o("active_color", "#123456") },
	func(ied string) *pyjson.Object { return o("state_mapping", direct("nope", 8)) },
	func(ied string) *pyjson.Object { return o("state_mapping", direct(ied, 8)) },
	func(ied string) *pyjson.Object { return o("state_mapping", direct(ied, 9)) },
	func(ied string) *pyjson.Object { return o("state_mapping", direct(ied, 7)) },
	func(ied string) *pyjson.Object { return o("control_binding", o("ied_id", ied, "ref", "LD0/CSWI4.Pos")) },
	func(ied string) *pyjson.Object { return o("control_binding", o("ied_id", ied, "ref", "nope")) },
	func(ied string) *pyjson.Object {
		return o("relay_statuses", []any{o("id", "a", "label", "A", "source", o("ied_id", ied, "ref", signalRef(3)))})
	},
	func(string) *pyjson.Object { return o("relay_statuses", []any{}) },
	func(string) *pyjson.Object { return o("state_mapping", o("mode", "contacts_to_dbpos")) },
}

// pythonInput is node_data's {"nodes": [... updated ...], "edges": edges}, built apart
// from the code under test.
func pythonInput(current *pyjson.Object, nodeID string, changes *pyjson.Object) *pyjson.Object {
	nodes := []any{}
	for _, item := range items(get(current, "nodes")) {
		node := item.(*pyjson.Object)
		if get(node, "id") == nodeID {
			copied, data := pyjson.NewObject(), pyjson.NewObject()
			for _, key := range node.Keys() {
				copied.Set(key, get(node, key))
			}
			for _, source := range []*pyjson.Object{object(get(node, "data")), changes} {
				for _, key := range source.Keys() {
					data.Set(key, get(source, key))
				}
			}
			node = copied.Set("data", data)
		}
		nodes = append(nodes, node)
	}
	return o("nodes", nodes, "edges", get(current, "edges"))
}

func get(object *pyjson.Object, key string) any {
	value, _ := object.Get(key)
	return value
}

func outcome(diagram *pyjson.Object, err error) string {
	if err != nil {
		if contractErr, ok := err.(*ContractError); ok {
			details, _ := pyjson.Dumps(contractErr.Details)
			return "contract " + string(details)
		}
		return fmt.Sprintf("%T %v", err, err)
	}
	data, _ := pyjson.Dumps(diagram)
	return string(data)
}

// TestNodeDataDiagramMatchesFullValidation: the incremental node_data diagram equals a
// full validate_diagram of Python's input for random valid and invalid changes, and never
// writes to the published diagram.
func TestNodeDataDiagramMatchesFullValidation(t *testing.T) {
	ieds := testIEDs()
	current, err := ValidateDiagram(testDiagram(), ieds)
	if err != nil {
		t.Fatal(err)
	}
	index := NewIndex()
	random := rand.New(rand.NewSource(7))
	nodes := items(get(current, "nodes"))
	compared, failures := 0, 0
	for step := 0; step < 3000; step++ {
		node := object(nodes[random.Intn(len(nodes))])
		id := text(node, "id")
		ied := fmt.Sprintf("ied%d", random.Intn(3))
		change := testChanges[random.Intn(len(testChanges))](ied)
		before, _ := pyjson.Dumps(current)
		want := outcome(ValidateDiagram(pythonInput(current, id, change), ieds))
		fast, fastErr := NodeDataDiagram(current, id, change, index.For(ieds), step%2 == 0)
		got := outcome(fast, fastErr)
		after, _ := pyjson.Dumps(current)
		compared++
		if got != want || string(before) != string(after) {
			failures++
			if failures < 5 {
				t.Errorf("step %d node %s change %v:\n got  %.400s\n want %.400s", step, id, outcome(change, nil), got, want)
			}
			continue
		}
		if fastErr == nil {
			current = fast
			nodes = items(get(current, "nodes"))
		}
	}
	t.Logf("%d node_data changes compared, %d differ", compared, failures)
}

// TestIndexLookupMatchesLinear: the indexed lookup answers like the scan, including
// duplicate refs, missing refs and non-object items.
func TestIndexLookupMatchesLinear(t *testing.T) {
	list := []any{"x", o("ref", "a", "n", int64(1))}
	for k := 0; k < 100; k++ {
		list = append(list, o("ref", fmt.Sprintf("r%d", k%60), "n", int64(k)))
	}
	list = append(list, o("n", int64(-1)), o("ref", int64(5)))
	ieds := []*pyjson.Object{o("id", "a", "signals", list)}
	indexed, linear := NewIndex().For(ieds), LinearLookup(ieds)
	for _, ref := range []string{"", "a", "r0", "r59", "r60", "nope", "5"} {
		if indexed.Entry(list, ref) != linear.Entry(list, ref) {
			t.Errorf("ref %q differs", ref)
		}
	}
}
