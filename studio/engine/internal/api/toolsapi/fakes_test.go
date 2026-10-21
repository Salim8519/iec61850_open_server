package toolsapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/explorer"
	"studio-engine/internal/tools/scan"
)

type fakeProjects struct {
	mu      sync.Mutex
	ieds    []*pyjson.Object
	section *pyjson.Object
	panel   []any
}

func (p *fakeProjects) IEDs() []*pyjson.Object { p.mu.Lock(); defer p.mu.Unlock(); return p.ieds }
func (p *fakeProjects) Gateway() *pyjson.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.section
}
func (p *fakeProjects) SaveGateway(config *pyjson.Object) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.section = config
	return nil
}
func (p *fakeProjects) TestPanel() []any { p.mu.Lock(); defer p.mu.Unlock(); return p.panel }
func (p *fakeProjects) SaveTestPanel(entries []any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.panel = entries
	return nil
}

// fakeRuntime is stream C's coordinator as the routes see it.
type fakeRuntime struct {
	state string
}

func (r *fakeRuntime) RunState() string                             { return r.state }
func (r *fakeRuntime) Signal(string, string) (service.Signal, bool) { return service.Signal{}, false }
func (r *fakeRuntime) Endpoint(string) (string, any, error) {
	return "", nil, apperr.ValueError("IED is not running")
}
func (r *fakeRuntime) Emit(string, *pyjson.Object)           {}
func (r *fakeRuntime) Subscribe(func(service.Change)) func() { return func() {} }
func (r *fakeRuntime) RunningSequences() int                 { return 0 }

type panelRuntime struct{ *fakeRuntime }

func (r panelRuntime) Stopped(fn func() error) error {
	if r.state != "stopped" {
		return apperr.New("Stop the simulation before editing its structure")
	}
	return fn()
}

// server mounts the area's routes like stream A's mux will (method + pattern).
func server(t *testing.T, deps Deps) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, route := range New(deps).Routes() {
		mux.Handle(route.Method+" "+route.Pattern, route.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func newDeps(projects *fakeProjects, runtime *fakeRuntime, executables map[string]string) Deps {
	if projects.section == nil {
		projects.section = gateway.EmptyGateway()
	}
	manager := service.NewManager(projects, runtime, nil, executables["gateway"], nil)
	master := service.NewMaster(projects, nil, executables["master"], nil)
	manager.Converter = &service.Converter{Master: master}
	return Deps{Gateway: manager, Master: master, Projects: projects, Runtime: panelRuntime{runtime},
		Explorer: explorer.NewService(executables["live"], nil, nil, nil, nil),
		Scanner:  scan.NewService(executables["live"], nil, nil, nil)}
}

// call sends a request and returns the status and the exact body bytes.
func call(t *testing.T, srv *httptest.Server, method, path, body string) (int, string) {
	t.Helper()
	request, _ := http.NewRequest(method, srv.URL+"/api/v1"+path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(data)
}

func decode(t *testing.T, text string) *pyjson.Object {
	t.Helper()
	value, err := pyjson.Decode([]byte(text))
	if err != nil {
		t.Fatalf("not JSON: %q", text)
	}
	object, _ := value.(*pyjson.Object)
	return object
}
