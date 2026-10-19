package projectapi

import (
	"net/http"

	"studio-engine/internal/project"
)

// getPreferences is GET /preferences: the guided-tour/UI preference document.
func (a *Area) getPreferences(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, 200, project.Preferences(a.deps.Preferences.Load()))
	return nil
}

// putPreferences is PUT /preferences with request.get_json(silent=True) as the changes.
func (a *Area) putPreferences(w http.ResponseWriter, r *http.Request) error {
	changes, _ := getJSON(r, true)
	stored, err := a.deps.Preferences.Update(changes, project.UpdatePreferences)
	if err != nil {
		return err
	}
	writeJSON(w, 200, project.Preferences(stored))
	return nil
}
