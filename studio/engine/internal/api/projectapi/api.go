// Package projectapi serves the project, SCL, IED file/removal and preferences routes of
// api/projects.py, api/scl.py, api/ieds.py and api/preferences.py with Python's exact
// paths, bodies and error texts.
package projectapi

import (
	"io"
	"net/http"
	"sync"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/project/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/settings"
)

// Runtime is what these routes need from the simulation coordinator (stream C).
type Runtime interface {
	// State is coordinator.state ("stopped", "running", ...).
	State() string
	// RunningIEDs is sorted(coordinator.active).
	RunningIEDs() []string
	// CycleLock is coordinator.cycle_lock (IED removal runs under it).
	CycleLock() sync.Locker
	// Emit publishes a bus event, e.g. ieds_removed {ied_ids, count}.
	Emit(kind string, payload *pyjson.Object)
}

// Startup reports startup recovery's result for GET /project (nil while none).
type Startup interface {
	Result() (*pyjson.Object, error)
}

// Deps are the services behind the routes.
type Deps struct {
	Projects    *service.Service
	Runtime     Runtime
	Startup     Startup
	Preferences *settings.Preferences
}

// Area is the projectapi route set.
type Area struct{ deps Deps }

// New returns the area for deps.
func New(deps Deps) *Area { return &Area{deps: deps} }

type handler func(w http.ResponseWriter, r *http.Request) error

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		drainBody(r)
		writeError(w, err)
	}
}

// drainBody reads what is left of a refused request (up to Flask's 100 MiB limit) before the
// answer is written. Go's server reads at most 256 KiB of an unread body and then closes the
// connection, so a client still sending a large upload got a TCP reset instead of the JSON
// error (change 0186); Werkzeug read the body to the end.
func drainBody(r *http.Request) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, io.LimitReader(r.Body, httpapi.MaxContentLength))
	}
}

// Routes implements api.Area.
func (a *Area) Routes() []api.Route {
	route := func(method, pattern string, h handler) api.Route {
		return api.Route{Method: method, Pattern: "/api/v1" + pattern, Handler: h}
	}
	return []api.Route{
		{Method: "GET", Pattern: "/api/v1/project", Handler: handler(a.getProject), Polled: true},
		route("PUT", "/project/diagram", a.putDiagram),
		route("POST", "/project/save", a.saveProject),
		route("GET", "/project/download", a.downloadProject),
		route("POST", "/project/open", a.openProject),
		route("POST", "/scl/validate", a.validateSCL),
		route("POST", "/scl/preview", a.previewSCL),
		route("POST", "/scl/import", a.importSCL),
		route("POST", "/ieds/template", a.template),
		route("GET", "/ieds/{ied_id}/source", a.source),
		route("GET", "/ieds/{ied_id}/data-classes", a.dataClasses),
		route("GET", "/diagram/goose-ieds", a.gooseIEDs),      // Go-only (ADR-0112)
		route("GET", "/ieds/{ied_id}/deadbands", a.deadbands), // Go-only (ADR-0118)
		route("PUT", "/ieds/{ied_id}/deadbands", a.setDeadbands),
		route("POST", "/ieds/{ied_id}/export-scl", a.exportSCL),
		route("GET", "/ieds/{key}/removal-preview", a.removalPreview),
		route("DELETE", "/ieds/{key}", a.removeIED),
		route("POST", "/ieds/removal-preview", a.bulkPreview),
		route("POST", "/ieds/remove", a.bulkRemove),
		route("PUT", "/ieds/{key}/settings", a.iedSettings),
		route("PUT", "/scenarios", a.putScenarios),
		route("GET", "/project/substation-sources", a.substationSources),
		route("POST", "/project/diagram/from-scl/preview", a.diagramPreview),
		route("GET", "/signal-flow/sources", a.flowSources),
		route("GET", "/signal-flow", a.flowGraph),
		route("GET", "/signal-flow/edges/{edge_id}.csv", a.edgeCSV),
		route("GET", "/preferences", a.getPreferences),
		route("PUT", "/preferences", a.putPreferences),
	}
}

// requireStopped is api/context.require_stopped.
func (a *Area) requireStopped() error {
	if a.deps.Runtime != nil && a.deps.Runtime.State() != "stopped" {
		return apperr.New("Stop the simulation before editing its structure")
	}
	return nil
}

func (a *Area) runtimeState() (string, []any) {
	if a.deps.Runtime == nil {
		return "stopped", []any{}
	}
	running := []any{}
	for _, id := range a.deps.Runtime.RunningIEDs() {
		running = append(running, id)
	}
	return a.deps.Runtime.State(), running
}
