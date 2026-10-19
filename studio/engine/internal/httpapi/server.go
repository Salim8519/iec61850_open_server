// Package httpapi is the Go engine's HTTP front: the Python app's request pipeline
// (loopback/origin 403 → session 401 → license 402 → startup wait → route), Flask's
// session cookie, Flask/Werkzeug error bodies, static frontend files, and the routes the
// area packages export as []api.Route.
package httpapi

import (
	"net/http"
	"time"

	"studio-engine/internal/api"
)

// LicenseGate is the license check (stream F implements it; nil never enforces).
type LicenseGate interface {
	// Enforced is LicenseService.enforce (true only in the packaged app).
	Enforced() bool
	// Licensed is is_licensed() (cached evaluation).
	Licensed() bool
}

// StartupGate is the background startup recovery (nil means ready).
type StartupGate interface {
	// Wait blocks until recovery finished or timeout passed; false on timeout.
	Wait(timeout time.Duration) bool
	// Err is the recovery error, or nil.
	Err() error
}

// Config holds what create_app received.
type Config struct {
	// Token is LAUNCH_TOKEN; SecretKey signs the session cookie.
	Token     string
	SecretKey string
	// AdminOrigin is the dev frontend URL (--frontend-url, trailing "/" removed); empty in
	// the packaged app, which then requires same-origin requests.
	AdminOrigin string
	License     LicenseGate
	Startup     StartupGate
	// StartupWait is how long engineering requests wait for recovery (300 s if zero).
	StartupWait time.Duration
	// Frontend serves "/" and "/assets/..." (nil: the build is not available).
	Frontend *Frontend
	Routes   []api.Route
}

// Server is the http.Handler of the Go engine.
type Server struct {
	config Config
	signer signer
	router router
	now    func() time.Time
}

// New builds the server; it fails when two routes collide or a pattern is malformed.
func New(config Config) (*Server, error) {
	if config.StartupWait == 0 {
		config.StartupWait = 300 * time.Second
	}
	s := &Server{config: config, signer: newSigner(config.SecretKey), now: time.Now}
	own := []api.Route{
		{Method: http.MethodPost, Pattern: "/api/v1/session", Handler: http.HandlerFunc(s.establishSession)},
		{Method: http.MethodGet, Pattern: "/", Handler: http.HandlerFunc(s.index)},
		{Method: http.MethodGet, Pattern: "/assets/{filename...}", Handler: http.HandlerFunc(s.asset)},
	}
	for _, r := range append(own, config.Routes...) {
		if err := s.router.add(r); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// SetClock replaces the time source used for cookie timestamps (tests).
func (s *Server) SetClock(now func() time.Time) { s.now = now }

// ServeHTTP runs the pipeline in the Python order. Handlers may panic with an error
// (*apperr.Error, *HTTPError or any other) to answer exactly as Flask's error handlers.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if recovered == http.ErrAbortHandler {
				panic(recovered)
			}
			err, _ := recovered.(error)
			WriteError(w, err)
		}
	}()
	if rejectBadURI(w, r) {
		return
	}
	normalizePath(r)
	if s.protect(w, r) || s.waitForStartup(w, r) {
		return
	}
	found := s.router.resolve(r)
	switch {
	case found.err != nil:
		WriteError(w, found.err)
	case found.redirect != "":
		writeRedirect(w, r, found.redirect)
	case found.route == nil:
		writeOptions(w, found.options)
	default:
		found.route.handler.ServeHTTP(w, r)
	}
}

// Handle adapts a handler that returns an error: a non-nil error is answered by WriteError.
func Handle(handler func(w http.ResponseWriter, r *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			WriteError(w, err)
		}
	})
}
