package runtimeapi

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// A point panel (ADR-0102) is served by the existing POST /equipment/<node>/test route: no new
// route, so the Flask route table is unchanged. A stopped run answers in plain words with 400.
func TestPointPanelUsesTheTestRoute(t *testing.T) {
	server, _, memory := newServer(t, 21320)
	diagram := pyval.Map(memory.Current(), "diagram")
	ied := pyval.Str(pyval.List(memory.Current(), "ieds")[0].(*pyjson.Object), "id")
	panel := pyval.Obj("id", "pp", "type", "equipment", "position", pyval.Obj("x", int64(0), "y", int64(0)),
		"data", pyval.Obj("kind", "POINT_PANEL", "label", "Panel", "rotation", int64(0), "ied_id", ied, "points", []any{}))
	diagram.Set("nodes", append(pyval.List(diagram, "nodes"), panel))
	status, body := call(t, server, "POST", "/equipment/pp/test", "application/json", `{"action":"healthy"}`)
	if status != 400 || !strings.Contains(body, "Run the IEDs to change points") {
		t.Fatalf("stopped panel: %d %s", status, body)
	}
	status, body = call(t, server, "POST", "/equipment/nope/test", "application/json", `{"action":"healthy"}`)
	if status != 400 || !strings.Contains(body, "Unknown equipment") {
		t.Fatalf("unknown node: %d %s", status, body)
	}
}
