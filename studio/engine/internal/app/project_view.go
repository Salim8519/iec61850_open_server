package app

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/api/projectapi"
	"studio-engine/internal/httpapi"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/settings"
)

// projectRoutes are api/projects.py, scl.py, ieds.py, preferences.py and the design routes.
// GET /project adds the runtime-only IED fields; opening a project or removing IEDs drops
// the runtime values of the IEDs that were replaced or removed (Python's values lived on
// the project's signal dicts and went with them).
func (e *Engine) projectRoutes() []api.Route {
	area := projectapi.New(projectapi.Deps{Projects: e.Projects, Runtime: e.Runtime, Startup: e.Recovery,
		Preferences: settings.NewPreferences(e.options.DataDir)})
	routes := area.Routes()
	for i, route := range routes {
		switch route.Method + " " + route.Pattern {
		case "GET /api/v1/project":
			routes[i].Handler = http.HandlerFunc(e.getProject)
		case "POST /api/v1/project/open":
			routes[i].Handler = e.dropValues(route.Handler, true)
		case "DELETE /api/v1/ieds/{key}", "POST /api/v1/ieds/remove":
			routes[i].Handler = e.dropValues(route.Handler, false)
		}
	}
	return routes
}

// getProject is GET /project: ProjectService.snapshot() of IED dicts that carried the
// runtime's "state" and "goose_state" (supervisor.py set them, popped goose_state on stop),
// plus runtime, running_ieds and startup_recovery.
func (e *Engine) getProject(w http.ResponseWriter, r *http.Request) {
	body := e.Projects.Snapshot()
	overlay := e.Runtime.IedStates()
	if raw, found := body.Get("ieds"); found && len(overlay) > 0 {
		list, _ := raw.([]any)
		for _, item := range list {
			ied, _ := item.(*pyjson.Object)
			if ied == nil {
				continue
			}
			id, _ := ied.Get("id")
			text, _ := id.(string)
			state, known := overlay[text]
			if !known {
				continue
			}
			if state.State != "" {
				ied.Set("state", state.State)
			}
			if state.HasGoose {
				ied.Set("goose_state", state.GooseState)
			} else {
				ied.Delete("goose_state")
			}
		}
	}
	running := []any{}
	for _, id := range e.Runtime.RunningIEDs() {
		running = append(running, id)
	}
	var recovered any
	if result, _ := e.Recovery.Result(); result != nil {
		recovered = result
	}
	body.Set("runtime", e.Runtime.State()).Set("running_ieds", running).Set("startup_recovery", recovered)
	httpapi.WriteJSON(w, http.StatusOK, body)
}

// dropValues resets, once the wrapped request answered 200, the runtime values of the IEDs
// it removed, or with replaced set of every IED before and after (a project was opened:
// Python then showed the values stored in the archive).
func (e *Engine) dropValues(next http.Handler, replaced bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		before := iedObjects(e.Projects.Current())
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if recorder.status != http.StatusOK {
			return
		}
		after := iedObjects(e.Projects.Current())
		for id := range before {
			if _, kept := after[id]; replaced || !kept {
				e.Runtime.ResetValues(id)
			}
		}
		if replaced {
			for id := range after {
				if _, seen := before[id]; !seen {
					e.Runtime.ResetValues(id)
				}
			}
		}
	})
}

// iedObjects maps IED id to its (immutable) project object.
func iedObjects(project *pyjson.Object) map[string]*pyjson.Object {
	result := map[string]*pyjson.Object{}
	raw, _ := project.Get("ieds")
	list, _ := raw.([]any)
	for _, item := range list {
		if ied, ok := item.(*pyjson.Object); ok {
			id, _ := ied.Get("id")
			if text, isText := id.(string); isText {
				result[text] = ied
			}
		}
	}
	return result
}

// statusRecorder remembers the status a handler answered.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}
