package liveapi

import (
	"net/http"

	"studio-engine/internal/acquisition"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/acquisition/signalindex"
	"studio-engine/internal/pyjson"
)

// runtimeValues reads point values from the runtime value table of a running IED.
type runtimeValues struct {
	values acquisition.Values
	iedID  string
}

func (v runtimeValues) Value(ref string) (any, bool) {
	value, _, found := v.values.Get(v.iedID, ref)
	return value, found
}

// listSignals is GET /ieds/<ied_id>/signals: the paged search over the prebuilt index.
func (s *Service) listSignals(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("ied_id")
	ied, err := s.Deps.Projects.FindIED(id)
	if err != nil {
		writeError(w, err)
		return
	}
	var values signalindex.ValueReader
	if s.Deps.Values.Has(id) {
		values = runtimeValues{values: s.Deps.Values, iedID: id}
	}
	query := signalindex.ParseQuery(signalindex.ParseArgs(r.URL.RawQuery))
	result, err := s.signals.For(ied).Search(query, values)
	if err != nil {
		writeError(w, err)
		return
	}
	if values != nil {
		s.overlay(id, pyv.GetList(result, "items"))
	}
	writeJSON(w, http.StatusOK, result)
}

// overlay puts the runtime "value" and "live_state" on each listed signal: Python's index
// entries are the signal dicts themselves, so a running IED lists them with both keys.
func (s *Service) overlay(iedID string, items []any) {
	for _, item := range pyv.Objects(items) {
		value, state, found := s.Deps.Values.Get(iedID, pyv.GetStr(item, "ref"))
		if !found {
			continue
		}
		item.Set("value", value)
		if state != nil {
			item.Set("live_state", state)
		}
	}
}

// testMode is PUT /ieds/<key>/test-mode.
func (s *Service) testMode(w http.ResponseWriter, r *http.Request) {
	value := silentJSON(r)
	if !pyv.Truthy(value) { // request.get_json(silent=True) or {}
		value = pyjson.NewObject()
	}
	result, err := s.Modes.Set(r.PathValue("key"), value)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
