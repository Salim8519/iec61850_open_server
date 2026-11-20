package projectapi

import (
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
)

// Auto save on/off (ADR-0139, Go-only): the File menu switch. The choice is remembered in
// preferences.json (autosave_off) and applied by the project service.

func (a *Area) autoSaveState() *pyjson.Object {
	return pyjson.NewObject().Set("on", !a.deps.Projects.ManualSave()).Set("unsaved", a.deps.Projects.Unsaved())
}

// saved tells Deps.Saved (the push stream) that {on, unsaved} may have moved.
func (a *Area) saved() {
	if a.deps.Saved != nil {
		a.deps.Saved()
	}
}

// getAutoSave is GET /project/autosave: {on, unsaved}.
func (a *Area) getAutoSave(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, 200, a.autoSaveState())
	return nil
}

// putAutoSave is PUT /project/autosave {on}. Turning it on saves the unsaved changes first; the
// choice is stored only once the project service took it.
func (a *Area) putAutoSave(w http.ResponseWriter, r *http.Request) error {
	body, _ := getJSON(r, true)
	table, _ := body.(*pyjson.Object)
	var on any
	if table != nil {
		on, _ = table.Get("on")
	}
	wanted, isBool := on.(bool)
	if !isBool {
		return apperr.ValueError("Auto save must be on or off")
	}
	err := a.deps.Projects.SetAutoSave(wanted)
	a.saved() // turning it on may have saved before failing; the answer is read again either way
	if err != nil {
		return err
	}
	off := map[bool]int64{true: 0, false: 1}[wanted]
	if _, err := a.deps.Preferences.Update(pyjson.NewObject().Set("autosave_off", off), project.UpdatePreferences); err != nil {
		return err
	}
	writeJSON(w, 200, a.autoSaveState())
	return nil
}
