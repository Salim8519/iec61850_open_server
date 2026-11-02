package projectapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Undo/Redo and New/Close routes (ADR-0135): status, a step back and forward, a named new
// project, a refused bad name, Close, and the running guard.
func TestHistoryAndNewProjectRoutes(t *testing.T) {
	mux, runtime := newArea(t)
	call := func(method, path, body string) (int, string) {
		request := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		return recorder.Code, recorder.Body.String()
	}
	if code, body := call("GET", "/project/history", ""); code != 200 || !strings.Contains(body, `"can_undo":false`) {
		t.Fatalf("empty status %d %s", code, body)
	}
	if code, body := call("POST", "/ieds/template", `{"name": "BayController"}`); code != 200 {
		t.Fatalf("template %d %s", code, body)
	}
	if _, body := call("GET", "/project/history", ""); !strings.Contains(body, `"undo":"IED change"`) {
		t.Fatalf("status after import %s", body)
	}
	if code, body := call("POST", "/project/undo", ""); code != 200 || !strings.Contains(body, `"can_redo":true`) {
		t.Fatalf("undo %d %s", code, body)
	}
	if code, body := call("POST", "/project/undo", ""); code != 400 || !strings.Contains(body, "nothing to undo") {
		t.Fatalf("second undo %d %s", code, body)
	}
	if code, _ := call("POST", "/project/redo", ""); code != 200 {
		t.Fatalf("redo %d", code)
	}
	if code, body := call("POST", "/project/new", `{"name": "  North yard  "}`); code != 200 ||
		!strings.Contains(body, `"name":"North yard"`) || !strings.Contains(body, `"ieds":[]`) {
		t.Fatalf("new %d %s", code, body)
	}
	if code, _ := call("POST", "/project/new", `{"name": 5}`); code != 400 {
		t.Fatalf("bad name %d", code)
	}
	if code, body := call("POST", "/project/close", ""); code != 200 || !strings.Contains(body, "Untitled substation") {
		t.Fatalf("close %d %s", code, body)
	}
	if _, body := call("GET", "/project", ""); !strings.Contains(body, `"startup_recovery":null`) {
		t.Fatal("a closed project still shows the startup recovery notice")
	}
	runtime.state = "running"
	if code, _ := call("POST", "/project/close", ""); code == http.StatusOK {
		t.Fatal("close while running")
	}
}
