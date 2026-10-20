package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"studio-engine/internal/gateway/service"
	"studio-engine/internal/testutil"
)

// TestComposedPushWiring: every push topic has its GET route, /capabilities answers behind
// the same security pipeline as /events, and a project change reaches the gateway
// (ProjectChanged, which replaced the gateway's 10 s sweep).
func TestComposedPushWiring(t *testing.T) {
	engine := composeAt(t, testutil.TempDir(t), testutil.TempDir(t))
	t.Cleanup(func() { _ = engine.Close() })
	if len(engine.PushMissing) > 0 {
		t.Fatalf("push topics without a GET route: %v", engine.PushMissing)
	}
	capabilities := mustCall(t, engine, "GET", "/api/v1/capabilities", nil)
	if push, _ := field(capabilities, "push").(int64); push != 1 {
		t.Fatalf("capabilities: %s", pyjsonText(capabilities))
	}
	if topics, _ := field(capabilities, "topics").([]any); len(topics) == 0 {
		t.Fatalf("capabilities list no topic: %s", pyjsonText(capabilities))
	}
	for _, path := range []string{"/api/v1/push?client=tab-1", "/api/v1/capabilities"} {
		if got, want := anonymous(engine, path), anonymous(engine, "/api/v1/events"); got != want {
			t.Errorf("%s without a session answered %d, /api/v1/events %d", path, got, want)
		}
	}

	seen := make(chan string, 64)
	engine.Gateway.OnChange(func(name string) {
		engine.Push.Changed(name)
		select {
		case seen <- name:
		default:
		}
	})
	mustCall(t, engine, "POST", "/api/v1/ieds/template", obj("name", "BayController"))
	deadline := time.After(5 * time.Second)
	for {
		select {
		case name := <-seen:
			if name == service.ChangeGatewayPoints {
				return
			}
		case <-deadline:
			t.Fatal("project_changed did not reach Gateway.ProjectChanged")
		}
	}
}

// anonymous is the status of a GET without a session or token.
func anonymous(engine *Engine, path string) int {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr, request.Host = "127.0.0.1:50000", "127.0.0.1:8765"
	recorder := httptest.NewRecorder()
	engine.Handler.ServeHTTP(recorder, request)
	return recorder.Code
}
