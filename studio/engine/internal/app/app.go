// Package app composes the Go engine: the event bus, the area routes and the HTTP server.
// It holds no behaviour of its own (composition only).
package app

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/events"
	"studio-engine/internal/httpapi"
)

// Config is what create_app received, plus the areas other workstreams provide.
type Config struct {
	DataDir     string
	FrontendDir string // dist folder or frontend.pak; empty: no frontend build
	Token       string
	SecretKey   string
	AdminOrigin string // dev frontend URL, empty in the packaged app
	License     httpapi.LicenseGate
	Startup     httpapi.StartupGate
	// Areas are the ported route groups; every other Python route answers 501.
	Areas []api.Area
	// Bus is the event bus areas publish on (a new one when nil).
	Bus *events.Bus
}

// App is a composed engine.
type App struct {
	Bus     *events.Bus
	Handler http.Handler
}

// New composes the engine.
func New(config Config) (*App, error) {
	bus := config.Bus
	if bus == nil {
		bus = events.NewBus(events.Capacity)
	}
	var frontend *httpapi.Frontend
	if config.FrontendDir != "" {
		loaded, err := httpapi.LoadFrontend(config.FrontendDir)
		if err != nil {
			return nil, err
		}
		frontend = loaded
	}
	areas := append([]api.Area{events.Area{Bus: bus}}, config.Areas...)
	var routes []api.Route
	served := map[[2]string]bool{}
	for _, area := range areas {
		for _, route := range area.Routes() {
			routes = append(routes, route)
			served[[2]string{route.Method, route.Pattern}] = true
		}
	}
	for _, entry := range pythonRoutes {
		if !served[entry] {
			routes = append(routes, api.Route{Method: entry[0], Pattern: entry[1], Handler: notPorted})
		}
	}
	server, err := httpapi.New(httpapi.Config{Token: config.Token, SecretKey: config.SecretKey,
		AdminOrigin: config.AdminOrigin, License: config.License, Startup: config.Startup,
		Frontend: frontend, Routes: routes})
	if err != nil {
		return nil, err
	}
	return &App{Bus: bus, Handler: server}, nil
}

// notPorted answers routes of the Python app that the Go preview does not serve yet.
var notPorted = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	httpapi.WriteError(w, &httpapi.HTTPError{Code: http.StatusNotImplemented,
		Description: "The Go engine preview does not provide this action yet. Use the Python engine for it."})
})
