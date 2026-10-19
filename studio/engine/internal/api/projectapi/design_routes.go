package projectapi

import (
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
)

// Routes of api/network.py (IED settings), api/scenarios.py (saved scenarios),
// api/sld.py and api/signal_flow.py.

// iedSettings is PUT /ieds/{key}/settings: under the cycle lock, stopped only.
func (a *Area) iedSettings(w http.ResponseWriter, r *http.Request) error {
	defer a.lockCycle()()
	if err := a.requireStopped(); err != nil {
		return err
	}
	raw, _ := getJSON(r, true)
	if !truthy(raw) {
		raw = pyjson.NewObject()
	}
	key := r.PathValue("key")
	result, err := a.deps.Projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		return project.UpdateIED(draft, key, raw)
	})
	if err != nil {
		return err
	}
	writeJSON(w, 200, result)
	return nil
}

// putScenarios is PUT /scenarios.
func (a *Area) putScenarios(w http.ResponseWriter, r *http.Request) error {
	if err := a.requireStopped(); err != nil {
		return err
	}
	raw, err := getJSON(r, false)
	if err != nil {
		return err
	}
	if err := a.deps.Projects.Scenarios(raw); err != nil {
		return err
	}
	writeJSON(w, 200, pyjson.NewObject().Set("ok", true))
	return nil
}

// substationSources is GET /project/substation-sources.
func (a *Area) substationSources(w http.ResponseWriter, _ *http.Request) error {
	sources, err := project.SubstationSources(a.deps.Projects.Project())
	if err != nil {
		return err
	}
	writeJSON(w, 200, pyjson.NewObject().Set("sources", sources))
	return nil
}

// diagramPreview is POST /project/diagram/from-scl/preview {"source_id", "mode"}. A body
// that is not exactly those two strings is the ValueError "Preview requires source_id and
// mode" (Python raised TypeError, a 500, for a non-empty list or a number; the UI never
// sends one).
func (a *Area) diagramPreview(w http.ResponseWriter, r *http.Request) error {
	raw, _ := getJSON(r, true)
	data := object(raw)
	if data == nil {
		data = pyjson.NewObject()
	}
	source, sourceOK := get(data, "source_id").(string)
	mode, modeOK := get(data, "mode").(string)
	if data.Len() != 2 || !sourceOK || !modeOK {
		return apperr.ValueError("Preview requires source_id and mode")
	}
	current, revision := a.deps.Projects.State()
	result, err := project.PreviewDiagram(current, revision, source, mode)
	if err != nil {
		return err
	}
	writeJSON(w, 200, result)
	return nil
}

// flowSource is api/signal_flow._source: the "source" argument or the first stored file.
func (a *Area) flowSource(r *http.Request, current *pyjson.Object) (string, error) {
	if source := r.URL.Query().Get("source"); source != "" {
		return source, nil
	}
	sources := project.FlowSources(current)
	if len(sources) == 0 {
		return "", apperr.ValueError("Import an SCL file first")
	}
	return get(sources[0].(*pyjson.Object), "id").(string), nil
}

// flowSources is GET /signal-flow/sources.
func (a *Area) flowSources(w http.ResponseWriter, _ *http.Request) error {
	writeJSON(w, 200, pyjson.NewObject().Set("sources", project.FlowSources(a.deps.Projects.Project())))
	return nil
}

// flowGraph is GET /signal-flow?source=&ied=.
func (a *Area) flowGraph(w http.ResponseWriter, r *http.Request) error {
	current := a.deps.Projects.Project()
	source, err := a.flowSource(r, current)
	if err != nil {
		return err
	}
	result, err := project.FlowGraph(current, source, r.URL.Query().Get("ied"))
	if err != nil {
		return err
	}
	writeJSON(w, 200, result)
	return nil
}

// edgeCSV is GET /signal-flow/edges/{edge_id}.csv: Response(text, mimetype="text/csv")
// with Content-Disposition attachment.
func (a *Area) edgeCSV(w http.ResponseWriter, r *http.Request) error {
	current := a.deps.Projects.Project()
	source, err := a.flowSource(r, current)
	if err != nil {
		return err
	}
	text, filename, err := project.EdgeCSV(current, source, r.PathValue("edge_id"))
	if err != nil {
		return err
	}
	header := w.Header()
	header.Set("Content-Type", "text/csv; charset=utf-8")
	header.Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(200)
	_, _ = w.Write([]byte(text))
	return nil
}
