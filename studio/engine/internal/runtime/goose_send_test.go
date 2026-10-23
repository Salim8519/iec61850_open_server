package runtime

import (
	"reflect"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// GOOSE send from the SLD (ADR-0112) on the fake workers: one action is one sample batch (so the
// native worker raises stNum once), refusals in plain words, and logged events with one
// correlation id. BayA publishes GoCB1 with XCBR1.Pos, XCBR1.Loc, XSWI1.Loc and GGIO1.AnIn1.

const sendBlock = "BayACTRL/LLN0$GO$GoCB1"

var sendMembers = []any{"XCBR1$ST$Pos$stVal", "XCBR1$ST$Loc$stVal", "XSWI1$ST$Loc$stVal", "GGIO1$MX$AnIn1$mag$f"}

func publishGoose(ied *pyjson.Object, prefix string) {
	refs := []any{prefix + "/XCBR1.Pos.stVal", prefix + "/XCBR1.Loc.stVal", prefix + "/XSWI1.Loc.stVal", prefix + "/GGIO1.AnIn1.mag.f"}
	datasets := append(pyval.List(ied, "datasets"), pyval.Obj("name", "GoDS", "members", sendMembers,
		"ref", prefix+"/LLN0$GoDS", "fcda_refs", refs))
	ied.Set("datasets", datasets).Set("goose", pyval.Obj("publishes", []any{pyval.Obj("name", "GoCB1",
		"gocb_ref", prefix+"/LLN0$GO$GoCB1", "dataset", "GoDS", "dataset_ref", prefix+"/LLN0$GoDS", "go_id", "GoCB1",
		"app_id", int64(1), "conf_rev", int64(1), "members", sendMembers)}, "subscribes", []any{}))
}

func sendNode(id, kind string, data ...any) any {
	fields := pyval.Obj("kind", kind, "label", id, "rotation", int64(0), "state_mapping", nil, "control_binding", nil,
		"relay_statuses", []any{})
	for index := 0; index+1 < len(data); index += 2 {
		fields.Set(data[index].(string), data[index+1])
	}
	return pyval.Obj("id", id, "type", "equipment", "position", pyval.Obj("x", int64(0), "y", int64(0)), "data", fields)
}

func sendLink(ied, prefix, mode string, refs ...string) *pyjson.Object {
	link := pyval.Obj("ied_id", ied, "gocb_ref", prefix+"/LLN0$GO$GoCB1", "mode", mode, "one", nil, "open", nil, "closed", nil)
	if mode == "contacts" {
		return link.Set("open", prefix+refs[0]).Set("closed", prefix+refs[1])
	}
	return link.Set("one", prefix+refs[0])
}

type batchLog struct {
	mu      sync.Mutex
	batches [][]item
}

// countBatches wraps the one batch write of a send for this test.
func countBatches(t *testing.T) *batchLog {
	log := &batchLog{}
	original := gooseWrite
	gooseWrite = func(c *Coordinator, iedID string, items []item, source string) (*pyjson.Object, error) {
		log.mu.Lock()
		log.batches = append(log.batches, append([]item(nil), items...))
		log.mu.Unlock()
		return original(c, iedID, items, source)
	}
	t.Cleanup(func() { gooseWrite = original })
	return log
}

// gooseRun has BayA simulated and BayB real (or simulated but not started when realB is false).
func gooseRun(t *testing.T, portBase int64, realB bool) (*Coordinator, *MemoryBus, string) {
	t.Helper()
	project, a, b := fakeProject(t, 21, portBase, false)
	publishGoose(projectIeds(project)[0], "BayACTRL")
	publishGoose(projectIeds(project)[1], "BayBCTRL")
	if realB {
		projectIeds(project)[1].Set("role", "real")
	}
	nodes := append(pyval.List(pyval.Map(project, "diagram"), "nodes"),
		sendNode("g-cb", "CB", "goose_link", sendLink(a, "BayACTRL", "contacts", "/XCBR1.Loc.stVal", "/XSWI1.Loc.stVal")),
		sendNode("g-ds", "DS", "goose_link", sendLink(a, "BayACTRL", "one", "/XCBR1.Pos.stVal")),
		sendNode("g-src", "SOURCE", "goose_link", sendLink(a, "BayACTRL", "one", "/GGIO1.AnIn1.mag.f")),
		sendNode("g-real", "DS", "goose_link", sendLink(b, "BayBCTRL", "one", "/XCBR1.Pos.stVal")),
		sendNode("g-ied", "IED", "ied_id", a), sendNode("g-plain", "CB"))
	pyval.Map(project, "diagram").Set("nodes", nodes)
	bus := &MemoryBus{}
	c := New(Options{NativeDir: fakeNativeDir(t), DataDir: testutil.TempDir(t), Bus: bus, Project: NewMemoryProject(project)})
	t.Cleanup(c.Close)
	return c, bus, a
}

func send(c *Coordinator, node string, pairs ...any) (*pyjson.Object, error) {
	return c.SendGoose(node, pyval.Obj(pairs...))
}

func TestGooseSendWritesOneBatchPerAction(t *testing.T) {
	c, bus, a := gooseRun(t, 26100, true)
	if _, err := send(c, "g-cb", "action", "open"); err == nil || !strings.Contains(err.Error(), "Run the IEDs to send GOOSE") {
		t.Fatalf("stopped: %v", err)
	}
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	log := countBatches(t)
	answer, err := send(c, "g-cb", "action", "open")
	if err != nil || pyval.Str(answer, "correlation_id") == "" || pyval.Str(answer, "gocb_ref") != sendBlock {
		t.Fatalf("contacts: %v %v", answer, err)
	}
	if len(log.batches) != 1 || !reflect.DeepEqual(log.batches[0], []item{{ref: "BayACTRL/XCBR1.Loc.stVal", value: true},
		{ref: "BayACTRL/XSWI1.Loc.stVal", value: false}}) {
		t.Fatalf("both contacts must go in one batch: %+v", log.batches)
	}
	for ref, want := range map[string]any{"BayACTRL/XCBR1.Loc.stVal": true, "BayACTRL/XSWI1.Loc.stVal": false} {
		if value, _ := c.Value(a, ref); value != want {
			t.Fatalf("%s holds %v", ref, value)
		}
	}
	if _, err := send(c, "g-cb", "action", "closed"); err != nil {
		t.Fatal(err)
	}
	if _, err := send(c, "g-ds", "action", "closed"); err != nil {
		t.Fatal(err)
	}
	if value, _ := c.Value(a, "BayACTRL/XCBR1.Pos.stVal"); value != int64(2) {
		t.Fatalf("Dbpos closed = 2, got %v", value)
	}
	if _, err := send(c, "g-src", "action", "value", "value", 12.5); err != nil {
		t.Fatal(err)
	}
	if _, err := send(c, "g-ied", "gocb_ref", sendBlock, "ref", "BayACTRL/XCBR1.Pos.stVal", "value", int64(1)); err != nil {
		t.Fatal(err)
	}
	if value, _ := c.Value(a, "BayACTRL/XCBR1.Pos.stVal"); value != int64(1) {
		t.Fatalf("the IED component wrote %v", value)
	}
	if len(log.batches) != 5 {
		t.Fatalf("five sends, five batches: %d", len(log.batches))
	}
	var kinds []string
	var ids []string
	for _, event := range bus.Events() {
		if kind := pyval.Str(event, "kind"); strings.HasPrefix(kind, "goose_send_") {
			kinds, ids = append(kinds, kind), append(ids, pyval.Str(event, "correlation_id"))
			if pyval.Str(event, "ied_id") != a || pyval.Str(event, "gocb_ref") != sendBlock || pyval.Str(event, "node_id") == "" {
				t.Fatalf("event without node, IED or block: %v", event)
			}
		}
	}
	if len(kinds) != 10 || kinds[0] != "goose_send_requested" || kinds[1] != "goose_send_applied" || ids[0] != ids[1] || ids[0] == ids[2] {
		t.Fatalf("one requested and one applied per send, sharing one id: %v %v", kinds, ids)
	}
	for _, event := range bus.Events() {
		if pyval.Str(event, "kind") == "goose_send_applied" {
			applied, _ := pyjson.Dumps(pyval.Map(event, "applied"))
			if want := `{"BayACTRL/XCBR1.Loc.stVal": true, "BayACTRL/XSWI1.Loc.stVal": false}`; strings.TrimSpace(string(applied)) != want {
				t.Fatalf("the first applied event names what the IED then held: %s", applied)
			}
			break
		}
	}
}

func TestGooseSendRefusesWithPlainWords(t *testing.T) {
	c, bus, a := gooseRun(t, 26200, true)
	if err := c.Start([]string{a}); err != nil {
		t.Fatal(err)
	}
	log := countBatches(t)
	cases := []struct {
		node  string
		pairs []any
		want  string
	}{
		{"g-real", []any{"action", "open"}, "BayB is a real device. ScadaBench only watches its GOOSE and never sends for it."},
		{"g-plain", []any{"action", "open"}, "g-plain has no GOOSE link"},
		{"nope", []any{"action", "open"}, "Unknown equipment"},
		{"g-cb", []any{"action", "on"}, "This switch sends Open or Closed."},
		{"g-ds", []any{"action", "value", "value", int64(1)}, "is a position: send Open or Closed"},
		{"g-src", []any{"action", "value", "value", "12"}, "enter a number"},
		{"g-ied", []any{"gocb_ref", "BayACTRL/LLN0$GO$Other", "ref", "BayACTRL/XCBR1.Pos.stVal", "value", int64(1)}, "does not send this GOOSE message"},
		{"g-ied", []any{"gocb_ref", sendBlock, "ref", "BayACTRL/XCBR1.Pos.q", "value", int64(1)}, "is not part of the GOOSE message"},
	}
	for _, item := range cases {
		if _, err := send(c, item.node, item.pairs...); err == nil || !strings.Contains(err.Error(), item.want) {
			t.Errorf("%s %v: %v, want %q", item.node, item.pairs, err, item.want)
		}
	}
	if len(log.batches) != 0 {
		t.Fatalf("a refused send writes nothing: %+v", log.batches)
	}
	for _, event := range bus.Events() {
		if strings.HasPrefix(pyval.Str(event, "kind"), "goose_send_") {
			t.Fatalf("a refusal before the write logs no request: %v", event)
		}
	}
}

func TestGooseSendRefusesAnIEDThatIsNotRunning(t *testing.T) {
	c, _, a := gooseRun(t, 26300, false)
	if err := c.Start([]string{a}); err != nil { // BayA runs, BayB does not
		t.Fatal(err)
	}
	if _, err := send(c, "g-real", "action", "open"); err == nil || err.Error() != "BayB is not running. Run this IED first." {
		t.Fatalf("an IED outside the run: %v", err)
	}
	if err := c.Stop(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := send(c, "g-ds", "action", "open"); err == nil || !strings.Contains(err.Error(), "Run the IEDs to send GOOSE") {
		t.Fatalf("after Stop: %v", err)
	}
}
