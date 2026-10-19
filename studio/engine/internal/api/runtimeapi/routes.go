package runtimeapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
	"studio-engine/internal/runtime/pyval"
)

// Area serves the runtime routes over one Coordinator.
type Area struct {
	Runtime *runtime.Coordinator
	// Project reads the project tree (GET /project/test-panel lists its entries).
	Project runtime.Project
}

// Routes implements api.Area with the Python paths under /api/v1.
func (a Area) Routes() []api.Route {
	route := func(method, pattern string, h handler, polled bool) api.Route {
		return api.Route{Method: method, Pattern: "/api/v1" + pattern, Handler: serve(h), Polled: polled}
	}
	return []api.Route{
		route("POST", "/runtime/{action}", a.command, false),
		route("POST", "/runtime/ieds/{ied_id}/{action}", a.iedCommand, false),
		route("GET", "/equipment/states", a.states, false),
		route("POST", "/equipment/{node_id}/operate", a.operate, false),
		route("POST", "/equipment/{node_id}/test", a.test, false),
		route("GET", "/scenarios/runs", a.runs, true),
		route("POST", "/scenarios/{key}/start", a.startScenario, false),
		route("POST", "/signals/update", a.update, false),
		route("POST", "/signals/source", a.source, false),
		route("POST", "/signals/pulse", a.pulse, false),
		route("POST", "/signals/release-override", a.release, false),
		route("POST", "/controls/operate", a.controlOperate, false),
		route("GET", "/project/test-panel", a.testPanel, true),
	}
}

func (a Area) runtimeState() (any, int, error) {
	return pyval.Obj("state", a.Runtime.State(), "running", toAny(a.Runtime.Running())), http.StatusOK, nil
}

func toAny(items []string) []any {
	values := make([]any, len(items))
	for i, item := range items {
		values[i] = item
	}
	return values
}

func unknownAction() (any, int, error) {
	return pyval.Obj("error", "Unknown runtime action"), http.StatusNotFound, nil
}

func (a Area) command(r *http.Request) (any, int, error) {
	var err error
	switch r.PathValue("action") {
	case "start":
		err = a.Runtime.Start(nil)
	case "stop":
		err = a.Runtime.Stop(nil)
	case "pause":
		err = a.Runtime.Pause()
	default:
		return unknownAction()
	}
	if err != nil {
		return nil, 0, err
	}
	return a.runtimeState()
}

func (a Area) iedCommand(r *http.Request) (any, int, error) {
	ids := []string{r.PathValue("ied_id")}
	var err error
	switch r.PathValue("action") {
	case "start":
		err = a.Runtime.Start(ids)
	case "stop":
		err = a.Runtime.Stop(ids)
	default:
		return unknownAction()
	}
	if err != nil {
		return nil, 0, err
	}
	return a.runtimeState()
}

func (a Area) states(*http.Request) (any, int, error) {
	states, energization := a.Runtime.EquipmentStates()
	return pyval.Obj("states", states, "energization", energization), http.StatusOK, nil
}

// operate is POST /equipment/<node_id>/operate.
func (a Area) operate(r *http.Request) (any, int, error) {
	data, ok := getJSONSilent(r).(*pyjson.Object)
	if !ok {
		return nil, 0, apperr.ValueError("Equipment operation must be a JSON object")
	}
	action := ""
	if value, present := data.Get("action"); present && value != nil {
		action = text(value)
	}
	result, err := a.Runtime.Operate(r.PathValue("node_id"), pyval.Get(data, "value"), action)
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}

func (a Area) test(r *http.Request) (any, int, error) {
	data, ok := getJSONSilent(r).(*pyjson.Object)
	if !ok {
		return nil, 0, apperr.ValueError("Equipment test must be a JSON object")
	}
	result, err := a.Runtime.EquipmentTest(r.PathValue("node_id"), pyval.Get(data, "action"),
		pyval.Get(data, "value"), pyval.Get(data, "row"))
	if err != nil {
		return nil, 0, err
	}
	return result, http.StatusOK, nil
}

func (a Area) runs(*http.Request) (any, int, error) {
	return a.Runtime.ScenarioRuns(), http.StatusOK, nil
}

func (a Area) startScenario(r *http.Request) (any, int, error) {
	if err := a.Runtime.StartScenario(r.PathValue("key")); err != nil {
		return nil, 0, err
	}
	return pyval.Obj("ok", true), http.StatusOK, nil
}
