package liveapi

import (
	"net/http"

	"studio-engine/internal/acquisition/health"
	"studio-engine/internal/acquisition/pyv"
)

// health is GET /health: station health plus the app version.
func (s *Service) health(w http.ResponseWriter, r *http.Request) {
	result := s.Live.StationHealth()
	result.Set("app_version", s.Version)
	writeJSON(w, http.StatusOK, result)
}

// healthRefresh is POST /health/refresh: read the settings again, then the health result
// with {"failures": {ied_id: message}} (no app_version, as in Python).
func (s *Service) healthRefresh(w http.ResponseWriter, r *http.Request) {
	failures := s.Live.RefreshHealth()
	writeJSON(w, http.StatusOK, pyv.Merge(pyv.O("failures", failures), s.Live.StationHealth()))
}

func (s *Service) healthCSV(w http.ResponseWriter, r *http.Request) {
	writeDownload(w, "text/csv", "attachment; filename=station-health.csv", health.CSVText(s.Live.StationHealth()))
}

func (s *Service) healthReport(w http.ResponseWriter, r *http.Request) {
	project := s.Deps.Projects.Current()
	page := health.HTMLReport(s.Live.StationHealth(), pyv.PyStr(pyv.Get(project, "name")), s.Deps.Projects.RunID())
	writeDownload(w, "text/html", "attachment; filename=station-health.html", page)
}
