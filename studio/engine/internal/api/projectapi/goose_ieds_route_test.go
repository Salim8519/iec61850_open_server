package projectapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/settings"
	"studio-engine/internal/testutil"
)

func o(pairs ...any) *pyjson.Object {
	result := pyjson.NewObject()
	for index := 0; index < len(pairs); index += 2 {
		result.Set(pairs[index].(string), pairs[index+1])
	}
	return result
}

// GET /diagram/goose-ieds (ADR-0112): Go proposes the IED components "Add all GOOSE IEDs" adds.
func TestGooseIEDsRoute(t *testing.T) {
	dir := testutil.TempDir(t)
	ids := counterIDs()
	projects, err := service.New(service.Config{DataDir: dir, NewID: ids, Importer: &scl.Importer{NewID: ids}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = projects.Close() })
	result, err := projects.RecoverStartup()
	area := New(Deps{Projects: projects, Runtime: &fakeRuntime{}, Startup: startupResult{result, err},
		Preferences: settings.NewPreferences(dir)})
	mux := http.NewServeMux()
	for _, route := range area.Routes() {
		mux.Handle(route.Method+" "+muxPattern(route.Pattern), route.Handler)
	}
	ied := func(id string, publishes int) any {
		list := []any{}
		for index := 0; index < publishes; index++ {
			list = append(list, o("gocb_ref", id+"LD0/LLN0$GO$G"))
		}
		return o("id", id, "name", "IED_"+id, "source", "<SCL/>", "goose", o("publishes", list, "subscribes", []any{}))
	}
	if _, err := projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		draft.Set("ieds", []any{ied("a", 1), ied("b", 0), ied("c", 2)})
		draft.Set("diagram", o("nodes", []any{o("id", "cb", "type", "equipment", "position", o("x", int64(40), "y", int64(400)),
			"data", o("label", "CB", "kind", "CB"))}, "edges", []any{}))
		return draft, nil
	}); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest("GET", "/api/v1/diagram/goose-ieds", nil))
	want := `{"nodes":[{"ied_id":"a","name":"IED_a","position":{"x":40.0,"y":220.0}},` +
		`{"ied_id":"c","name":"IED_c","position":{"x":280.0,"y":220.0}}]}`
	if recorder.Code != 200 || strings.TrimSpace(recorder.Body.String()) != want {
		t.Fatalf("proposal: %d %s", recorder.Code, recorder.Body.String())
	}
}
