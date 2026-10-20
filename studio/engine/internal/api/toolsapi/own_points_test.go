package toolsapi

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// TestOwnPointRoutes: the Go-only own point routes (lane G) suggest a new point, keep own
// points through Propose and the point list save, and change a saved value while stopped.
func TestOwnPointRoutes(t *testing.T) {
	projects := &fakeProjects{}
	srv := server(t, newDeps(projects, &fakeRuntime{state: "stopped"}, map[string]string{}))
	status, text := call(t, srv, "POST", "/tools/gateway/points/new",
		`{"protocol": "iec104", "kind": "single", "points": [{"kind": "single", "address": 10001}]}`)
	if status != 200 || text != `{"point":{"address":10002,"description":"","id":"own:1","ied":"","kind":"single","ref":"","scale":1.0,"source":"manual","value":0}}`+"\n" {
		t.Fatalf("new point %d %s", status, text)
	}
	status, text = call(t, srv, "POST", "/tools/gateway/propose",
		`{"protocol": "iec104", "keep": [{"kind": "float", "source": "manual", "address": 30001, "id": "own:1", "description": "Level"}]}`)
	points, _ := decode(t, text).Get("points")
	kept := points.([]any)[0].(*pyjson.Object)
	if status != 200 || len(points.([]any)) != 1 || value(kept, "ied_name", nil) != "Own point" ||
		value(kept, "monitor_only", nil) != false {
		t.Fatalf("propose %d %s", status, text)
	}
	status, text = call(t, srv, "PUT", "/tools/gateway/points",
		`{"protocol": "iec104", "points": [{"kind": "float", "source": "manual", "address": 30001, "id": "own:1", "value": 3}]}`)
	if status != 200 || !strings.Contains(text, `"ied_name":"Own point"`) || !strings.Contains(text, `"value_text":"3"`) {
		t.Fatalf("save %d %s", status, text)
	}
	status, text = call(t, srv, "PUT", "/tools/gateway/points/value", `{"point_id": "own:1", "value": 7.25}`)
	if status != 200 || !strings.Contains(text, `"value":7.25`) || !strings.Contains(text, `"value_text":"7.25"`) {
		t.Fatalf("value %d %s", status, text)
	}
	status, text = call(t, srv, "PUT", "/tools/gateway/points/value", `{"point_id": "own:1", "value": "x"}`)
	if status != 400 || !strings.Contains(text, "The value was not changed: the value must be a number") {
		t.Fatalf("bad value %d %s", status, text)
	}
	status, text = call(t, srv, "PUT", "/tools/gateway/points/value", `{"point_id": "nope", "value": 1}`)
	if status != 400 || !strings.Contains(text, "Save point list first") {
		t.Fatalf("unknown point %d %s", status, text)
	}
	stored, _ := value(projects.Gateway(), "points", nil).(*pyjson.Object)
	list, _ := value(stored, "iec104", nil).([]any)
	if value(list[0].(*pyjson.Object), "value", nil) != 7.25 {
		t.Fatalf("stored %v", list)
	}
}
