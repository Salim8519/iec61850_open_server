package liveapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/pyjson"
)

// env is the Go engine wired to fakes that mirror the Python scenario's fakes.
type env struct {
	t        *testing.T
	projects *fakeProjects
	runtime  *fakeRuntime
	events   *recorder
	values   *fakeValues
	script   *scriptedWorkers
	service  *Service
	mux      *http.ServeMux
	handler  http.Handler // mux, or the composed server pipeline around the same routes
	composed bool
	decls    goosedom.Declarations
}

func loadScenario(t *testing.T) *pyjson.Object {
	data, err := os.ReadFile(filepath.Join(goldentest.Dir(), "routes_scenario.json"))
	if err != nil {
		t.Fatalf("run testdata/gen/gen_routes.py: %v", err)
	}
	decoded, err := pyjson.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	return pyv.Obj(decoded)
}

func newEnv(t *testing.T, doc *pyjson.Object) *env {
	folder := t.TempDir()
	for _, name := range []string{"studio_live.exe", "studio_goose.exe"} {
		if err := os.WriteFile(filepath.Join(folder, name), []byte("fake"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	now, _ := time.Parse(time.RFC3339, pyv.GetStr(doc, "now"))
	e := &env{t: t, projects: &fakeProjects{}, runtime: &fakeRuntime{state: "stopped"}, events: &recorder{},
		values: &fakeValues{}, script: newScript(pyv.GetList(doc, "script")), mux: http.NewServeMux()}
	monotonic, _ := pyv.Get(doc, "monotonic").(float64)
	deps := acquisition.Deps{LiveExecutable: filepath.Join(folder, "studio_live.exe"), Start: e.script.start,
		Projects: e.projects, Runtime: e.runtime, Events: e.events, Bus: e.events, Values: e.values,
		Samples: idle{}, Equipment: idle{}, States: idle{},
		Network: fakeNetwork{adapters: pyv.Objects(pyv.GetList(doc, "adapters")), npcap: pyv.GetObj(doc, "npcap")},
		Clock:   fixedClock{now: now.UTC(), monotonic: monotonic}}
	e.service = &Service{Deps: deps, Live: acquisition.NewManager(deps), Access: acquisition.NewMmsAccess(e.events, deps.Clock),
		Modes: acquisition.NewTestModes(deps), Version: pyv.GetStr(doc, "version"),
		Declarations: func(*pyjson.Object) (goosedom.Declarations, error) { return e.decls, nil }}
	for _, route := range e.service.Routes() {
		e.mux.Handle(route.Method+" "+route.Pattern, route.Handler)
	}
	e.handler = e.mux
	return e
}

// TestRoutesMatchPython replays the recorded Flask answers against the bare routes.
func TestRoutesMatchPython(t *testing.T) { replayRoutes(t, false) }

// TestComposedRoutesMatchPython (gap 6) replays them through the composed server's HTTP
// pipeline (httpapi.New with the session token, as app.Compose mounts these routes), with
// the same fakes: the session check, security and gates must not change a recorded answer.
func TestComposedRoutesMatchPython(t *testing.T) { replayRoutes(t, true) }

func replayRoutes(t *testing.T, composed bool) {
	doc := loadScenario(t)
	e := newEnv(t, doc)
	if composed {
		server, err := httpapi.New(httpapi.Config{Token: "token", SecretKey: "secret", Routes: e.service.Routes()})
		if err != nil {
			t.Fatal(err)
		}
		e.handler, e.composed = server, true
	}
	defer e.service.Live.Stop()
	counts := map[string]int{}
	defer func() { t.Logf("replayed steps %v", counts) }()
	for index, step := range pyv.Objects(pyv.GetList(doc, "steps")) {
		counts[pyv.GetStr(step, "step")]++
		switch pyv.GetStr(step, "step") {
		case "project":
			e.projects.project = pyv.GetObj(step, "project")
			e.projects.runID = pyv.GetStr(step, "run_id")
			e.decls = goosedom.DeclarationsFromJSON(pyv.GetObj(step, "declarations"))
		case "http":
			e.http(index, step)
		case "start":
			e.start(index, step)
		case "event":
			e.inject(index, step)
		case "stop":
			e.service.Live.Stop()
			e.compareEvents(index, "stop", step, false)
		}
	}
	if len(e.script.misses) > 0 {
		t.Errorf("native requests Python never made: %v", e.script.misses)
	}
}

func (e *env) http(index int, step *pyjson.Object) {
	var body *strings.Reader
	if text, ok := pyv.Get(step, "body").(string); ok {
		body = strings.NewReader(text)
	} else {
		body = strings.NewReader("")
	}
	request := httptest.NewRequest(pyv.GetStr(step, "method"), pyv.GetStr(step, "path"), body)
	if ctype := pyv.GetStr(step, "ctype"); ctype != "" {
		request.Header.Set("Content-Type", ctype)
	}
	if e.composed {
		request.RemoteAddr, request.Host = "127.0.0.1:50000", "127.0.0.1:8765"
		request.Header.Set("Authorization", "Bearer token")
	}
	e.events.take()
	recorder := httptest.NewRecorder()
	e.handler.ServeHTTP(recorder, request)
	name := pyv.GetStr(step, "method") + " " + pyv.GetStr(step, "path")
	wantStatus, _ := pyv.Int64(pyv.Get(step, "status"))
	if int64(recorder.Code) != wantStatus || recorder.Body.String() != goExpected(pyv.GetStr(step, "path"), pyv.GetStr(step, "response")) {
		e.t.Errorf("step %d %s: status %d want %d\n got: %s\nwant: %s", index, name, recorder.Code, wantStatus,
			clip(recorder.Body.String()), clip(goExpected(pyv.GetStr(step, "path"), pyv.GetStr(step, "response"))))
	}
	if got, want := recorder.Header().Get("Content-Type"), pyv.GetStr(step, "type"); got != want {
		e.t.Errorf("step %d %s: Content-Type %q want %q", index, name, got, want)
	}
	if got, want := recorder.Header().Get("Content-Disposition"), pyv.GetStr(step, "disposition"); got != want {
		e.t.Errorf("step %d %s: Content-Disposition %q want %q", index, name, got, want)
	}
	e.compareEvents(index, name, step, false)
}

func (e *env) start(index int, step *pyjson.Object) {
	e.events.take()
	e.runtime.state = "running"
	project := e.projects.Current()
	e.values.seed(project)
	if err := e.service.Live.Start(project, nil); err != nil {
		e.t.Fatalf("start: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		sessions := e.service.Live.Sessions()
		if len(sessions) > 0 && pyv.GetStr(sessions[0].Status(), "state") == "live" && e.script.readCount() >= 3 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	// The session's first live poll emits its `signal` asynchronously: under load it could
	// land after the step ended and be counted in the next one. Wait (bounded) until the
	// step's recorded number of events arrived; the comparison itself is unchanged.
	want, settle := len(expectedEvents(step)), time.Now().Add(10*time.Second)
	for time.Now().Before(settle) && e.events.count() < want {
		time.Sleep(5 * time.Millisecond)
	}
	e.compareEvents(index, "start", step, true)
}

func (e *env) inject(index int, step *pyjson.Object) {
	e.events.take()
	event := pyv.CopyObject(pyv.GetObj(step, "event"))
	delivered := true
	switch pyv.GetStr(step, "target") {
	case "goose":
		delivered = e.script.deliver("live", "goose_start", event)
	case "sv":
		delivered = e.script.deliver("live", "sv_start", event)
	case "engine":
		delivered = e.script.deliver("goose", "", event)
	case "mms":
		e.service.Access.Update(pyv.GetStr(step, "ied_id"), event)
	}
	if !delivered {
		e.t.Errorf("step %d: no worker for %s", index, pyv.GetStr(step, "target"))
	}
	e.compareEvents(index, "event "+pyv.GetStr(step, "target"), step, false)
}

// compareEvents checks the events since the step began; project_changed belongs to the
// project actor (stream B) and is not produced by these fakes.
func (e *env) compareEvents(index int, name string, step *pyjson.Object, sorted bool) {
	want := expectedEvents(step)
	var got []string
	for _, pair := range e.events.take() {
		got = append(got, pair[0]+" "+pair[1])
	}
	if sorted {
		sort.Strings(want)
		sort.Strings(got)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		e.t.Errorf("step %d %s events differ\n got: %s\nwant: %s", index, name,
			clip(strings.Join(got, "\n")), clip(strings.Join(want, "\n")))
	}
}

// expectedEvents are the step's recorded events as "kind payload" lines.
func expectedEvents(step *pyjson.Object) []string {
	var want []string
	for _, item := range pyv.GetList(step, "events") {
		pair := pyv.List(item)
		if pyv.Str(pair[0]) != "project_changed" {
			want = append(want, pyv.Str(pair[0])+" "+pyv.Str(pair[1]))
		}
	}
	return want
}

func clip(text string) string {
	if len(text) > 2500 {
		return text[:2500] + "..."
	}
	return text
}
