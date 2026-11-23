package app

import (
	"net/http"
	"time"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/api"
	"studio-engine/internal/api/liveapi"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/realcmd"
	"studio-engine/internal/runtime/ports"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/scl"
	"studio-engine/internal/tools/control"
)

// spareEngineDelay keeps the first spare GOOSE engine out of the application's own start.
const spareEngineDelay = 3 * time.Second

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
	// The first Run finds a GOOSE engine ready as well, started once the window has opened.
	time.AfterFunc(spareEngineDelay, e.Live.Engine.Prepare)
	live.manager = e.Live
	declarations := &project.GooseDeclarations{} // one cache per process
	if e.Control == nil { // shared with the Explorer and the gateway (wireTools)
		e.Control = control.NewClient(o.native("studio_control.exe"), nil)
	}
	commands := &realcmd.Service{FindIED: e.Projects.FindIED, Control: e.Control, Emit: e.Runtime.Emit,
		Extended: func(ied *pyjson.Object) []any {
			list, err := scl.CachedExtendedControls(pyval.Str(ied, "id"), pyval.Str(ied, "name"), pyval.Str(ied, "source"))
			if err != nil {
				return nil
			}
			return list
		},
		Sessions: func(id string) (realcmd.Session, error) {
			session, err := e.Live.Session(id)
			if err != nil || session == nil {
				return nil, err
			}
			return session, nil
		}}
	area := &liveapi.Service{Deps: deps, Live: e.Live, Access: acquisition.NewMmsAccess(a.Bus, deps.Clock), Commands: commands,
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

// Stop ends the Run's listeners and GOOSE engine, then readies a fresh engine for the next
// Run in the background (ADR-0133).
func (l *liveAdapter) Stop() {
	l.manager.Stop()
	l.manager.Engine.Prepare()
}

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
