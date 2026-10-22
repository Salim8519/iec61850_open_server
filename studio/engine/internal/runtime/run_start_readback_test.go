package runtime

import (
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Change 0176 (N1/B12): Run reads back every SLD row from the workers, not only the mapped
// equipment states. A worker restarts from its SCL values, so after Stop and Run the tap
// position must be the device's, not the previous run's.
func TestRunReadsBackTapRowsFromTheWorker(t *testing.T) {
	c, bus, a := startTap(t, 24100, "spc_pair", avrOn(132, 30), false, 7)
	if got, _ := c.readTap(readTapNode(mustNode(t, c, "t1"))); got != 7 {
		t.Fatalf("the tap was set to 7 in the first run, read %d", got)
	}
	if err := c.Stop(nil); err != nil {
		t.Fatal(err)
	}
	before := len(bus.Events())
	if err := c.Start(nil); err != nil {
		t.Fatal(err)
	}
	snapshot := false
	for _, event := range bus.Events()[before:] {
		if pyval.Str(event, "kind") == "signal" && pyval.Str(event, "source") == "initial_native_snapshot" &&
			pyval.Str(event, "ied_id") == a && pyval.Str(event, "ref") == fakeTap {
			snapshot = true
		}
	}
	if !snapshot {
		t.Fatal("Run must read the tap position back from the restarted worker")
	}
	if got, known := c.readTap(readTapNode(mustNode(t, c, "t1"))); known && got == 7 {
		t.Fatal("after Stop and Run the tap still shows the previous run's 7")
	}
}

func mustNode(t *testing.T, c *Coordinator, id string) *pyjson.Object {
	t.Helper()
	node, err := c.findNode(id)
	if err != nil {
		t.Fatal(err)
	}
	return node
}

func TestGoOnlyRowBindingsLeaveRelaysOut(t *testing.T) {
	relay := pyval.Obj("id", "r1", "data", pyval.Obj("kind", "RELAY", "relay_statuses", []any{
		pyval.Obj("id", "s1", "source", pyval.Obj("ied_id", "A", "ref", "A/PTRC1.Op.general"))}))
	list := pyval.Obj("id", "m1", "data", pyval.Obj("kind", "MEASUREMENT_LIST", "measurements", []any{
		pyval.Obj("id", "x", "source", pyval.Obj("ied_id", "A", "ref", "A/MMXU1.TotW.mag.f")),
		pyval.Obj("id", "y", "source", nil)}))
	project := pyval.Obj("diagram", pyval.Obj("nodes", []any{relay, list}))
	got := goOnlyRowBindings(project)
	if len(got) != 1 || got[0].Ref != "A/MMXU1.TotW.mag.f" {
		t.Fatalf("only the measurement row is read back here (relays keep Python parity): %v", got)
	}
}
