package acquisition

import (
	"studio-engine/internal/acquisition/health"
	"studio-engine/internal/acquisition/modes"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// StationHealth is acquisition/health_snapshot.station_health: the live sessions and the
// GOOSE/SV listeners combined into one result. It reads the immutable published project
// (no copies, no project lock) and never talks to a device.
func (m *Manager) StationHealth() *pyjson.Object {
	project := m.deps.Projects.Current()
	ieds := pyv.Objects(pyv.GetList(project, "ieds"))
	views := make([]*pyjson.Object, 0, len(ieds))
	keep := map[string]bool{}
	for _, ied := range ieds {
		views = append(views, modeCache.view(ied))
		keep[pyv.GetStr(ied, "id")] = true
	}
	modeCache.forget(keep)
	running := m.deps.Runtime.State() != "stopped"
	sessions := map[string]*pyjson.Object{}
	goose, sv := pyjson.NewObject(), pyjson.NewObject()
	if running {
		for _, session := range m.Sessions() {
			sessions[session.ID()] = session.Status()
		}
		goose = m.Goose.Snapshot()
		sv = m.SV.Snapshot()
	}
	result := health.BuildHealth(views, running, sessions, goose, m.deps.Clock.Now())
	result.Set("modes", modes.ModeBadges(views, sessions, pyv.OrObj(project, "diagram"), running))
	sources := []any{}
	if running {
		sources = modes.SimulationSources(goose, sv)
	}
	result.Set("simulation_sources", sources)
	return result
}

// RefreshHealth reads the report-block, GOOSE-block and mode settings again on each
// existing session; it returns {ied_id: error text} for the sessions that failed.
func (m *Manager) RefreshHealth() *pyjson.Object {
	failures := pyjson.NewObject()
	for _, session := range m.Sessions() {
		if _, err := session.RefreshHealth(); err != nil {
			failures.Set(session.ID(), err.Error())
		}
	}
	return failures
}
