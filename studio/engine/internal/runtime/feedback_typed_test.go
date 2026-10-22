package runtime

import (
	"testing"
	"time"

	"studio-engine/internal/runtime/pyval"
)

// A typed control (ADR-0104) applied in the worker reports a non-Boolean ctlVal ("higher" for a
// BSC step). It drives no switch symbol, but its new status must reach the drawing: the tap
// changer's tap row is published with the new position and no "feedback failed" event is raised
// (found in the 0173 browser run: the IED moved to 6, the drawing kept showing 5).
func TestTypedControlAppliedPublishesItsStatus(t *testing.T) {
	c, bus, a := startTap(t, 23950, "step", avrOn(132, 30), false, 9)
	c.onWorkerEvent(a, pyval.Obj("event", "control_applied", "ref", "BayACTRL/YLTC1.TapChg",
		"status_ref", fakeTap, "value", int64(10), "ctl_value", "higher", "timestamp", time.Now().UnixMilli()))
	published := func() bool {
		for _, event := range tapEvents(bus, "equipment_state") {
			if pyval.Str(event, "node_id") == "t1:tap" && pyval.Get(event, "value") == int64(10) {
				return true
			}
		}
		return false
	}
	deadline := time.Now().Add(3 * time.Second)
	for !published() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if !published() {
		t.Fatal("the tap row was not published with the applied position 10")
	}
	if failed := tapEvents(bus, "equipment_feedback_failed"); len(failed) != 0 {
		t.Fatalf("a typed control is not a failed feedback: %v", failed)
	}
}

