package toolsapi

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// smartPropose answers POST /tools/gateway/propose with the Go-only "mode" (ADR-0105):
// "replace" (the IED points are proposed again, the own points of "current" stay) or "add_new"
// (the list in "current" stays as it is, with the user's edits, and only missing points are added). Both
// fill names from the SCL, add each command's status point and pair their addresses.
// "ieds" (optional) are the ticked IED ids. The answer adds "summary" for the preview:
// one row per IED with what it offers and how much of that is new. Nothing is saved here.
func (a *Area) smartPropose(data *pyjson.Object, protocol string) (any, error) {
	mode := pytext.Str(value(data, "mode", ""))
	if mode != "replace" && mode != "add_new" {
		return nil, apperr.ValueError("Choose the mode: replace or add_new")
	}
	current, _ := value(data, "current", []any{}).([]any)
	options := addpoints.SmartOptions{AddNew: mode == "add_new", Current: current}
	if raw, ticked := value(data, "ieds", nil).([]any); ticked {
		options.Ticked = map[string]bool{}
		for _, id := range raw {
			options.Ticked[pytext.Str(id)] = true
		}
	}
	points, notes, summary, err := addpoints.Smart(a.deps.Projects.IEDs(), protocol, options)
	if err != nil {
		return nil, err
	}
	rows := make([]any, 0, len(summary))
	for _, item := range summary {
		rows = append(rows, pyjson.NewObject().Set("id", item.ID).Set("name", item.Name).Set("role", item.Role).
			Set("offered", int64(item.Offered)).Set("new", int64(item.New)).Set("ticked", item.Ticked))
	}
	return pyjson.NewObject().Set("points", a.named(points)).Set("notes", toStrings(notes)).Set("summary", rows), nil
}
