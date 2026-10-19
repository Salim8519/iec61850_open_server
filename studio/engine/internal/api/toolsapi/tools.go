package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

func truthy(value any) bool { return pytext.Truthy(value) }

func (a *Area) toolRoutes() []api.Route {
	d := a.deps
	return []api.Route{
		route("GET", "/activity", true, func(*http.Request) (any, error) { return a.activity(), nil }),
		route("GET", "/tools/memory", false, func(*http.Request) (any, error) { return d.Memory.Get(), nil }),
		route("PUT", "/tools/memory/find", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			choice, isText := value(data, "choice", nil).(string)
			if !isText || pytext.Len(choice) > 200 {
				return nil, apperr.ValueError("Send the chosen network as text")
			}
			return d.Memory.Update("find", pyjson.NewObject().Set("choice", choice))
		}),
		route("POST", "/tools/scan/preview", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			return d.Scanner.Preview(value(data, "range", ""), value(data, "port", int64(102)))
		}),
		route("POST", "/tools/scan/start", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			return d.Scanner.Start(value(data, "range", ""), value(data, "port", int64(102)))
		}),
		route("POST", "/tools/scan/stop", false, func(*http.Request) (any, error) { return d.Scanner.Stop(), nil }),
		route("GET", "/tools/scan", true, func(*http.Request) (any, error) { return d.Scanner.Status(), nil }),
	}
}

// activity is tools/activity.activity: what runs now outside the IED Run (never starts anything).
func (a *Area) activity() *pyjson.Object {
	d := a.deps
	status := d.Gateway.Status()
	opened, _ := value(status, "opened", nil).(*pyjson.Object)
	var port any
	if opened != nil {
		port = value(opened, "port", nil)
	}
	clients, _ := value(status, "clients", nil).([]any)
	state, probed, total := d.Scanner.Snapshot()
	master := pyjson.NewObject().Set("state", "disconnected").Set("protocol", "").Set("host", "").Set("points", int64(0))
	if d.Master != nil {
		current := d.Master.Status()
		master = pyjson.NewObject().Set("state", value(current, "state", nil)).
			Set("protocol", value(current, "protocol", nil)).Set("host", value(current, "host", nil)).
			Set("points", value(current, "points", nil))
	}
	return pyjson.NewObject().
		Set("gateway", pyjson.NewObject().Set("state", value(status, "state", nil)).
			Set("protocol", value(status, "protocol", nil)).Set("port", port).Set("clients", int64(len(clients)))).
		Set("explorer", d.Explorer.Activity()).
		Set("scan", pyjson.NewObject().Set("state", state).Set("probed", probed).Set("total", total)).
		Set("master", master).
		Set("sequences", int64(d.Runtime.RunningSequences()))
}
