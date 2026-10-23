package projectapi

import (
	"net/http"

	"studio-engine/internal/goosemsg"
	"studio-engine/internal/pyjson"
)

// gooseIEDs is GET /diagram/goose-ieds (Go-only, ADR-0112): the IED components "Add all GOOSE
// IEDs" places, decided in Go: every project IED that sends or receives GOOSE and has no component
// on the drawing yet, in one evenly spaced row above the drawing's top. The browser inserts them
// through its normal diagram edit, so the whole row is one Undo step.
func (a *Area) gooseIEDs(w http.ResponseWriter, r *http.Request) error {
	project := a.deps.Projects.Project()
	diagram, _ := get(project, "diagram").(*pyjson.Object)
	nodes, _ := get(diagram, "nodes").([]any)
	var ieds []*pyjson.Object
	list, _ := get(project, "ieds").([]any)
	for _, item := range list {
		if ied, ok := item.(*pyjson.Object); ok {
			ieds = append(ieds, ied)
		}
	}
	placements := []any{}
	for _, place := range goosemsg.Propose(nodes, ieds) {
		placements = append(placements, pyjson.NewObject().Set("ied_id", place.IEDID).Set("name", place.Name).
			Set("position", pyjson.NewObject().Set("x", place.X).Set("y", place.Y)))
	}
	writeJSON(w, http.StatusOK, pyjson.NewObject().Set("nodes", placements))
	return nil
}
