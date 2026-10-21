package app

import (
	"net/http"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/api"
	"studio-engine/internal/api/liveapi"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/ports"
	"studio-engine/internal/scl"
)

// wireLive builds LiveManager (real-IED sessions, GOOSE/SV listeners and the GOOSE engine)
// on the coordinator's ports, binds it to the runtime and returns the live routes.
// GET /mms/access reads the coordinator's client list: the runtime owns MMS access (it
// sees the worker summaries), acquisition's copy is never fed.
func (e *Engine) wireLive(live *liveAdapter) []api.Route {
	o := e.options
	a := ports.ForAcquisition(e.Runtime)
	deps := acquisition.Deps{LiveExecutable: o.native("studio_live.exe"), Start: a.Start,
		Projects: acquisitionProjects{e.Projects}, Runtime: a.Runtime, Events: a.Events, Bus: a.Bus,
		Values: a.Values, Samples: a.Samples, Equipment: a.Equipment, States: a.States,
		Network: systemNetwork{}, Clock: systemClock{}, Members: scl.SubscriptionMembers}
	e.Live = acquisition.NewManager(deps)
	live.manager = e.Live
	declarations := &project.GooseDeclarations{} // one cache per process
	area := &liveapi.Service{Deps: deps, Live: e.Live, Access: acquisition.NewMmsAccess(a.Bus, deps.Clock),
		Modes: acquisition.NewTestModes(deps), Version: o.Version,
		Declarations: func(p *pyjson.Object) (goosedom.Declarations, error) {
			value, err := declarations.For(p)
			if err != nil {
				return goosedom.Declarations{}, err
			}
			return goosedom.DeclarationsFromJSON(value), nil
		}}
	return replaceRoutes(area.Routes(), api.Route{Method: "GET", Pattern: "/api/v1/mms/access",
		Handler: http.HandlerFunc(e.mmsAccess), Polled: true})
}

// mmsAccess is GET /mms/access: jsonify(running=state != "stopped", ieds=snapshot).
func (e *Engine) mmsAccess(w http.ResponseWriter, r *http.Request) {
	httpapi.WriteJSON(w, http.StatusOK, pyjson.NewObject().Set("running", e.Runtime.State() != "stopped").
		Set("ieds", e.Runtime.MmsAccess()))
}

// liveAdapter is acquisition.Manager as runtime.Live (the run's ids arrive as a list there
// and as a set here; BusFor answers a map instead of an ordered object).
type liveAdapter struct{ manager *acquisition.Manager }

func (l *liveAdapter) Start(p *pyjson.Object, active []string) error {
	selected := make(map[string]bool, len(active))
	for _, id := range active {
		selected[id] = true
	}
	return l.manager.Start(p, selected)
}

func (l *liveAdapter) Connect(p *pyjson.Object, real []*pyjson.Object) error {
	return l.manager.Connect(p, real)
}

func (l *liveAdapter) Disconnect(iedID string) { l.manager.Disconnect(iedID) }

func (l *liveAdapter) Stop() { l.manager.Stop() }

func (l *liveAdapter) BusFor(ieds []*pyjson.Object) (map[string]string, error) {
	opened, err := l.manager.Engine.BusFor(ieds)
	if err != nil {
		return nil, err
	}
	buses := make(map[string]string, opened.Len())
	for _, id := range opened.Keys() {
		value, _ := opened.Get(id)
		buses[id], _ = value.(string)
	}
	return buses, nil
}

// replaceRoutes swaps the handlers of routes with the same method and pattern.
func replaceRoutes(routes []api.Route, replacements ...api.Route) []api.Route {
	out := make([]api.Route, len(routes))
	copy(out, routes)
	for _, replacement := range replacements {
		for i := range out {
			if out[i].Method == replacement.Method && out[i].Pattern == replacement.Pattern {
				out[i] = replacement
			}
		}
	}
	return out
}
