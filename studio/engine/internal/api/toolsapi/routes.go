package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/gateway/service"
	"studio-engine/internal/tools/explorer"
	"studio-engine/internal/tools/memory"
	"studio-engine/internal/tools/scan"
)

// Runtime is what saving the test panel and the activity dots need from the IED runtime.
// GET /project/test-panel belongs to internal/api/runtimeapi (one owner per route).
type Runtime interface {
	// Stopped holds the runtime's cycle lock while fn runs, after refusing with
	// StudioError "Stop the simulation before editing its structure" unless stopped.
	Stopped(fn func() error) error
	// RunningSequences counts scenario runs in state "running".
	RunningSequences() int
}

// Projects is the project service as the gateway and the test panel use it (stream B).
type Projects interface {
	service.Projects
	// SaveTestPanel stores the validated list, saves and announces the change; on a save
	// failure the previous list stays.
	SaveTestPanel(entries []any) error
}

// Deps are the services behind the routes; the composition (stream A) builds them.
type Deps struct {
	Explorer *explorer.Service
	Scanner  *scan.Service
	Memory   *memory.Store
	Gateway  *service.Manager
	Master   *service.Master
	Projects Projects
	Runtime  Runtime
}

// Area serves the routes of api/tools.py, api/gateway.py, api/gateway_master.py,
// api/test_panel.py (PUT /project/test-panel only) and api/traces.py.
type Area struct{ deps Deps }

// New returns the area for deps.
func New(deps Deps) *Area { return &Area{deps: deps} }

func route(method, path string, polled bool, fn func(*http.Request) (any, error)) api.Route {
	return api.Route{Method: method, Pattern: "/api/v1" + path, Handler: handle(fn), Polled: polled}
}

// Routes lists every route with Python's exact paths.
func (a *Area) Routes() []api.Route {
	var routes []api.Route
	routes = append(routes, a.explorerRoutes()...)
	routes = append(routes, a.toolRoutes()...)
	routes = append(routes, a.gatewayRoutes()...)
	routes = append(routes, a.pointRoutes()...)
	routes = append(routes, a.ownRoutes()...)
	routes = append(routes, a.masterRoutes()...)
	routes = append(routes, a.masterPointRoutes()...)
	routes = append(routes, a.panelRoutes()...)
	return routes
}
