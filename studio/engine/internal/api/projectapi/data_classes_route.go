package projectapi

import (
	"net/http"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// GoOnlyRoutes are the routes this area adds beyond Flask's (ADR-0095).
var GoOnlyRoutes = map[string]bool{"GET /api/v1/ieds/{ied_id}/data-classes": true}

// dataClasses is GET /ieds/{ied_id}/data-classes: {"classes": {object ref: cdc}} read from the
// IED's stored SCL. The frontend refines point kinds with it; older engines answer 404 and the
// frontend keeps the kinds it can tell from FC and basic type.
func (a *Area) dataClasses(w http.ResponseWriter, r *http.Request) error {
	ied, err := a.deps.Projects.FindIED(r.PathValue("ied_id"))
	if err != nil {
		return err
	}
	id, _ := get(ied, "id").(string)
	name, _ := get(ied, "name").(string)
	source, _ := get(ied, "source").(string)
	classes, err := scl.CachedDataClasses(id, name, source)
	if err != nil {
		return err
	}
	result := pyjson.NewObject()
	for ref, class := range classes {
		result.Set(ref, class)
	}
	writeJSON(w, 200, pyjson.NewObject().Set("classes", result))
	return nil
}
