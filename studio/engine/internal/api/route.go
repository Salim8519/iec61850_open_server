// Package api holds the contract every workstream uses to expose HTTP routes (ADR-0076):
// each area package returns its routes; the httpapi package mounts them behind the shared
// security order (403 → 401 → 402 → startup wait → route) and the Flask-compatible errors.
package api

import "net/http"

// Route is one HTTP route, written exactly as the Python blueprint path with Go 1.22
// patterns, for example {Method: "POST", Pattern: "/api/v1/equipment/{node_id}/operate"}.
type Route struct {
	Method  string
	Pattern string
	Handler http.Handler
	// Polled marks routes the frontend polls, so the startup wait and logging treat them
	// as Python does; the parity inventory lists them.
	Polled bool
}

// Area is implemented by every package that serves routes.
type Area interface {
	Routes() []Route
}
