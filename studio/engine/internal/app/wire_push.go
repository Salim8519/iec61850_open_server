package app

import (
	"studio-engine/internal/api"
	"studio-engine/internal/events"
	"studio-engine/internal/push"
	"studio-engine/internal/runtime"
)

// PushOnlyRoutes are the Go-only routes of the push stream (lane G): Python answers them
// 404, which is how the frontend knows to keep polling there.
var PushOnlyRoutes = map[string]bool{"GET /api/v1/capabilities": true, "GET /api/v1/push": true,
	"PUT /api/v1/push/topics": true}

// wirePush builds the push stream (one stream per tab, topics read through the exact GET
// handlers of the live, runtime and tools areas) and hooks the services' change
// notifications into it. Every project_changed reaches the gateway (ProjectChanged), which
// replaced the gateway's 10 s sweep.
func (e *Engine) wirePush(live, runtimeRoutes, tools []api.Route) []api.Route {
	topics, missing := push.Catalog(push.IndexRoutes(live, runtimeRoutes, tools), push.CatalogOptions{
		Running:  func() bool { return e.Runtime.State() != "stopped" },
		Scanning: e.Scanner.Scanning,
	})
	if e.PushMissing = missing; len(missing) > 0 {
		e.options.Logf("Push topics without a GET route: %v", missing)
	}
	e.Push = push.New(push.Options{
		Bus:    e.bus,
		Topics: topics,
		Values: func(notify func(iedID, ref string)) func() {
			return e.Runtime.Subscribe(func(change runtime.Change) { notify(change.IEDID, change.Ref) })
		},
		Listen: func(event *events.Event) {
			if event.Kind == "project_changed" {
				e.Gateway.ProjectChanged()
			}
		},
	})
	e.Gateway.OnChange(e.Push.Changed)
	e.Master.OnChange(e.Push.Changed)
	e.Explorer.OnChange(e.Push.Changed)
	e.Scanner.OnChange(e.Push.Changed)
	return e.Push.Routes()
}
