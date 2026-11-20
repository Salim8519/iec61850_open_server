package projectapi

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// Change 0224: File › Save project and the auto save switch tell Deps.Saved, so the push stream
// re-reads /project/autosave ({on, unsaved} moves without a project_changed); reads, edits and a
// refused switch value do not.
func TestSaveAndAutoSaveSwitchTellSaved(t *testing.T) {
	calls := 0
	mux, _ := newAreaSaved(t, func() { calls++ })
	call := func(method, path, body string) (int, string) {
		request := httptest.NewRequest(method, "/api/v1"+path, strings.NewReader(body))
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		recorder := httptest.NewRecorder()
		mux.ServeHTTP(recorder, request)
		return recorder.Code, recorder.Body.String()
	}
	if code, _ := call("GET", "/project/autosave", ""); code != 200 || calls != 0 {
		t.Fatalf("a read told Saved: %d %d", code, calls)
	}
	if code, body := call("PUT", "/project/autosave", `{"on": false}`); code != 200 || calls != 1 ||
		!strings.Contains(body, `"on":false`) {
		t.Fatalf("switch off: %d %s, %d calls", code, body, calls)
	}
	if code, _ := call("POST", "/ieds/template", `{"name": "BayController"}`); code != 200 || calls != 1 {
		t.Fatalf("an edit told Saved: %d calls", calls)
	}
	if _, body := call("GET", "/project/autosave", ""); !strings.Contains(body, `"unsaved":true`) {
		t.Fatalf("unsaved after an edit with auto save off: %s", body)
	}
	if code, _ := call("POST", "/project/save", ""); code != 200 || calls != 2 {
		t.Fatalf("save: %d, %d calls", code, calls)
	}
	if code, _ := call("PUT", "/project/autosave", `{"on": "yes"}`); code != 400 || calls != 2 {
		t.Fatalf("a refused value told Saved: %d, %d calls", code, calls)
	}
	if code, _ := call("PUT", "/project/autosave", `{"on": true}`); code != 200 || calls != 3 {
		t.Fatalf("switch on: %d, %d calls", code, calls)
	}
}
