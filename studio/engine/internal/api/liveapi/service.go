// Package liveapi serves the live-reads area of the API (Python blueprints live, goose_share,
// health, signals list and test_mode) as []api.Route with byte-identical Flask bodies.
package liveapi

import (
	"net/http"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/signalindex"
	"studio-engine/internal/api"
	"studio-engine/internal/pyjson"
)

// Declarations returns the merged GOOSE declarations of the project's stored SCL files
// (projects/goose_links._declarations: scl/goose_sources.read_goose_sources, cached per set
// of files). Stream B provides it; an empty project gives empty declarations.
type Declarations func(project *pyjson.Object) (goosedom.Declarations, error)

// Service holds what the live routes read.
type Service struct {
	Deps         acquisition.Deps
	Live         *acquisition.Manager
	Access       *acquisition.MmsAccess
	Modes        *acquisition.TestModes
	Declarations Declarations
	Version      string // studio_app.version.__version__

	signals signalindex.Cache
}

// Routes returns the area's routes with Python's exact paths.
func (s *Service) Routes() []api.Route {
	route := func(method, pattern string, polled bool, handler http.HandlerFunc) api.Route {
		return api.Route{Method: method, Pattern: "/api/v1" + pattern, Handler: handler, Polled: polled}
	}
	return []api.Route{
		route("GET", "/live", true, s.live),
		route("GET", "/live/{key}/comparison", false, s.comparison),
		route("GET", "/live/{key}/comparison.csv", false, s.comparisonCSV),
		route("POST", "/live/{key}/read", false, s.read),
		route("GET", "/goose", true, s.goose),
		route("GET", "/goose/links", true, s.gooseLinks),
		route("GET", "/mms/access", true, s.mmsAccess),
		route("GET", "/sv", true, s.sampledValues),
		route("GET", "/goose/share", true, s.sharePreview),
		route("PUT", "/goose/share", false, s.shareApply),
		route("POST", "/goose/conflicts", false, s.shareChoose),
		route("GET", "/health", true, s.health),
		route("POST", "/health/refresh", false, s.healthRefresh),
		route("GET", "/health.csv", false, s.healthCSV),
		route("GET", "/health/report.html", false, s.healthReport),
		route("GET", "/ieds/{ied_id}/signals", false, s.listSignals),
		route("PUT", "/ieds/{key}/test-mode", false, s.testMode),
		route("GET", "/ieds/{ied_id}/goose/messages", false, s.gooseMessages), // Go-only (ADR-0112)
		route("GET", "/goose/members", false, s.gooseMembers),
	}
}

func (s *Service) running() bool { return s.Deps.Runtime.State() != "stopped" }
