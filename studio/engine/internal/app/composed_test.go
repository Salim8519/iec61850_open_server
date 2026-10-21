package app

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/events"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/testutil"
)

// clickProject imports one BayController template through the routes, moves it to a
// loopback address of this test (127.67.0.x, port >= 20000) and draws one bound CB.
func clickProject(t testing.TB, engine *Engine) string {
	t.Helper()
	ids := field(mustCall(t, engine, "POST", "/api/v1/ieds/template", obj("name", "BayController")), "ieds").([]any)
	id := ids[0].(string)
	address, port := "127.67.0."+itoa(10+rand.Intn(200)), int64(20000+rand.Intn(20000))
	if _, err := engine.Projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		ied, err := project.FindIED(draft, id)
		if err == nil {
			ied.Set("ip", address).Set("port", port)
		}
		return ied, err
	}); err != nil {
		t.Fatal(err)
	}
	source := obj("ied_id", id, "ref", "BayControllerCTRL/XCBR1.Pos.stVal", "report_ref", nil)
	node := obj("id", "CB1", "type", "equipment", "position", obj("x", int64(100), "y", int64(40)),
		"data", obj("label", "CB 1", "kind", "CB", "state_mapping", obj("mode", "direct", "source", source),
			"control_binding", obj("ied_id", id, "ref", "BayControllerCTRL/XCBR1.Pos")))
	junction := obj("id", "J1", "position", obj("x", int64(0), "y", int64(0)), "data", obj("label", "J", "kind", "JUNCTION"))
	diagram := obj("nodes", []any{node, junction}, "edges", []any{obj("id", "w1", "source", "CB1", "target", "J1")})
	revision := field(mustCall(t, engine, "GET", "/api/v1/project", nil), "revision")
	mustCall(t, engine, "PUT", "/api/v1/project/diagram", obj("diagram", diagram, "revision", revision, "tab_id", "tab-1"))
	return id
}

func itoa(n int) string { return pyjsonText(int64(n)) }

// operateCB sends one canvas operate and returns the equipment_state event of the new
// position with the times of the answer and of the event.
func operateCB(t testing.TB, engine *Engine, closeIt bool) (time.Duration, time.Duration) {
	t.Helper()
	wanted := int64(1)
	if closeIt {
		wanted = 2
	}
	cursor := engine.Bus.Sequence()
	started := time.Now()
	mustCall(t, engine, "POST", "/api/v1/equipment/CB1/operate", obj("value", closeIt))
	done := time.Since(started)
	event, _ := waitEvent(engine.Bus, cursor, 5*time.Second, func(event *events.Event) bool {
		value, _ := event.Payload.Get("value")
		return event.Kind == "equipment_state" && field(event.Payload, "node_id") == "CB1" && value == wanted
	})
	if event == nil {
		t.Fatalf("no equipment_state %d for CB1 after the operate", wanted)
	}
	// The event's own time.time() stamp: when the symbol's event left the engine.
	symbol := time.Duration((event.Timestamp - float64(started.UnixNano())/1e9) * float64(time.Second))
	return done, symbol
}

// TestComposedClickPath: import, draw, Run, canvas CB operate (symbol event and saved
// position), GET /project with the runtime IED state, Stop, close, and a restart on the
// same folder restores the operated position.
func TestComposedClickPath(t *testing.T) {
	nativeDir := muxNative(t)
	dataDir := testutil.TempDir(t)
	engine := composeAt(t, dataDir, nativeDir)
	closed := false
	defer func() {
		if !closed {
			_ = engine.Close()
		}
	}()
	id := clickProject(t, engine)
	state := mustCall(t, engine, "POST", "/api/v1/runtime/start", nil)
	if field(state, "state") != "running" {
		t.Fatalf("runtime after start: %s", pyjsonText(state))
	}
	snapshot := mustCall(t, engine, "GET", "/api/v1/project", nil)
	ied := field(snapshot, "ieds").([]any)[0]
	if field(ied, "state") != "running" || field(snapshot, "runtime") != "running" {
		t.Fatalf("GET /project while running: %s", pyjsonText(snapshot))
	}
	for i, closeIt := range []bool{true, false, true} {
		done, symbol := operateCB(t, engine, closeIt)
		t.Logf("operate %d: answer %.1f ms, symbol %.1f ms", i, ms(done), ms(symbol))
	}
	node := engine.Projects.Current()
	position := field(firstNode(node), "data", "initial_position")
	if position != "closed" {
		t.Fatalf("operated position not saved: %v", position)
	}
	mustCall(t, engine, "POST", "/api/v1/runtime/stop", nil)
	if after := field(field(mustCall(t, engine, "GET", "/api/v1/project", nil), "ieds").([]any)[0], "state"); after != "stopped" {
		t.Fatalf("IED state after stop: %v", after)
	}
	if err := engine.Close(); err != nil {
		t.Fatal(err)
	}
	closed = true
	if _, err := os.Stat(filepath.Join(dataDir, autosave.JournalName)); !os.IsNotExist(err) {
		t.Fatal("the journal must be folded into autosave.scadabench on close")
	}
	reopened := composeAt(t, dataDir, nativeDir)
	defer reopened.Close()
	again := mustCall(t, reopened, "GET", "/api/v1/project", nil)
	if field(again, "startup_recovery", "status") != "restored" ||
		field(firstNode(reopened.Projects.Current()), "data", "initial_position") != "closed" ||
		field(field(again, "ieds").([]any)[0], "id") != id {
		t.Fatalf("restart did not restore the operated project: %s", pyjsonText(again))
	}
}

func firstNode(p *pyjson.Object) any {
	nodes, _ := field(p, "diagram", "nodes").([]any)
	if len(nodes) == 0 {
		return nil
	}
	return nodes[0]
}

func ms(d time.Duration) float64 { return float64(d.Microseconds()) / 1000 }
