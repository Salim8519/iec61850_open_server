package app

import (
	"strings"
	"testing"

	"studio-engine/internal/gateway/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
	"studio-engine/internal/testutil"
)

type fakeStates map[string]runtime.IEDOverlay

func (f fakeStates) IedStates() map[string]runtime.IEDOverlay { return f }

type fakeIEDs []*pyjson.Object

func (f fakeIEDs) IEDs() []*pyjson.Object                  { return f }
func (f fakeIEDs) Gateway() *pyjson.Object                 { return nil }
func (f fakeIEDs) SaveGateway(config *pyjson.Object) error { return nil }

// The gateway sees the runtime state Python kept on the IED dict; the project stays untouched.
func TestGatewayProjectsCarryRuntimeState(t *testing.T) {
	stored := obj("id", "a", "name", "A", "state", "stopped")
	other := obj("id", "b", "name", "B", "state", "stopped", "goose_state", "old")
	view := gatewayProjects{fakeIEDs{stored, other}, fakeStates{
		"a": {State: "running", GooseState: "publishing", HasGoose: true},
		"b": {State: ""},
	}}
	ieds := view.IEDs()
	if field(ieds[0], "state") != "running" || field(ieds[0], "goose_state") != "publishing" {
		t.Fatalf("running IED: %s", pyjsonText(ieds[0]))
	}
	if field(ieds[1], "state") != "stopped" || field(ieds[1], "goose_state") != nil {
		t.Fatalf("IED without a runtime state keeps the stored one, goose_state popped: %s", pyjsonText(ieds[1]))
	}
	if field(stored, "state") != "stopped" || field(other, "goose_state") != "old" {
		t.Fatal("the published project must never be changed")
	}
	if !service.IEDOnline(ieds[0], "running") || service.IEDOnline(ieds[1], "running") {
		t.Fatal("ied_online must follow the runtime state")
	}
}

// Real workers: with 4 simulated IEDs running, the gateway's points carry values (Python:
// "OFF", source_ok) and commands to a running simulated IED are not refused as "not running".
func TestComposedGatewaySeesRunningIEDs(t *testing.T) {
	nativeDir := muxNative(t)
	engine := composeAt(t, testutil.TempDir(t), nativeDir)
	defer engine.Close()
	id := clickProject(t, engine)
	proposal := mustCall(t, engine, "POST", "/api/v1/tools/gateway/propose", obj("protocol", "iec104"))
	points := field(proposal, "points").([]any)
	mustCall(t, engine, "PUT", "/api/v1/tools/gateway/points", obj("protocol", "iec104", "points", points))
	table := func() []any {
		return field(mustCall(t, engine, "GET", "/api/v1/tools/gateway/points?protocol=iec104", nil), "points").([]any)
	}
	for _, row := range table() {
		if online, has := row.(*pyjson.Object).Get("online"); has && online != false {
			t.Fatalf("stopped: %s", pyjsonText(row))
		}
	}
	mustCall(t, engine, "POST", "/api/v1/runtime/start", nil)
	defer mustCall(t, engine, "POST", "/api/v1/runtime/stop", nil)
	monitored, commands := 0, 0
	for _, raw := range table() {
		row := raw.(*pyjson.Object)
		if field(row, "ied") != id {
			continue
		}
		if kind, _ := field(row, "kind").(string); strings.HasPrefix(kind, "command_") {
			commands++
			if refusal := service.Refusal(row, engine.Gateway.IED(id), true); strings.Contains(refusal, "not running") {
				t.Fatalf("command refused while running: %s", refusal)
			}
			continue
		}
		monitored++
		if field(row, "online") != true || strings.HasPrefix(field(row, "value_text").(string), "no value") {
			t.Fatalf("running IED point without a value: %s", pyjsonText(row))
		}
	}
	if monitored == 0 || commands == 0 {
		t.Fatalf("expected monitored and command points, got %d and %d", monitored, commands)
	}
}
