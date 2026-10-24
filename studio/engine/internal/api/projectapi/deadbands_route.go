package projectapi

import (
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/deadband"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// deadbandSwitch is the running side of the switch (runtime.Coordinator).
type deadbandSwitch interface{ SetDeadbands(iedID string, on bool) }

// deadbands is GET /ieds/{ied_id}/deadbands: {enabled, points, applied, without_range}.
func (a *Area) deadbands(w http.ResponseWriter, r *http.Request) error {
	ied, err := a.deps.Projects.FindIED(r.PathValue("ied_id"))
	if err != nil {
		return err
	}
	writeJSON(w, 200, deadbandPoints(ied))
	return nil
}

func deadbandPoints(ied *pyjson.Object) *pyjson.Object {
	id, _ := get(ied, "id").(string)
	name, _ := get(ied, "name").(string)
	source, _ := get(ied, "source").(string)
	writable, _ := scl.CachedDeadbandWrites(id, name, source) // unreadable SCL: nothing writable
	return deadband.Points(ied, writable, get(ied, "scd_deadbands") == true)
}

// setDeadbands is PUT /ieds/{ied_id}/deadbands {"enabled": bool}: it saves the switch (stored
// only while on, so projects without it stay unchanged) and applies it to a running IED.
func (a *Area) setDeadbands(w http.ResponseWriter, r *http.Request) error {
	data, _ := getJSON(r, true)
	object, _ := data.(*pyjson.Object)
	on, ok := get(object, "enabled").(bool)
	if object == nil || !ok {
		return apperr.ValueError("Send {\"enabled\": true} or {\"enabled\": false}")
	}
	iedID := r.PathValue("ied_id")
	ied, err := a.deps.Projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		ied, err := project.FindIED(draft, iedID)
		if err != nil {
			return nil, err
		}
		if on {
			ied.Set("scd_deadbands", true)
		} else {
			ied.Delete("scd_deadbands")
		}
		return ied, nil
	})
	if err != nil {
		return err
	}
	if running, ok := a.deps.Runtime.(deadbandSwitch); ok {
		running.SetDeadbands(iedID, on)
	}
	writeJSON(w, 200, deadbandPoints(ied))
	return nil
}
