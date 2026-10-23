package projectapi

import (
	"errors"
	"net/http"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// errNotDict is Python's AttributeError on `request.get_json().get(...)` for non-dict
// JSON: an unhandled exception answered as the generic 500.
var errNotDict = errors.New("JSON payload is not an object")

const maxImport = 20

// validateSCL is POST /scl/validate (no stop check, as in Python).
func (a *Area) validateSCL(w http.ResponseWriter, r *http.Request) error {
	source, _, err := upload(r, "file")
	if err != nil {
		return err
	}
	verdict, err := scl.Validate(source, a.validator())
	if err != nil {
		return err
	}
	writeJSON(w, 200, verdict)
	return nil
}

func (a *Area) validator() scl.SchemaValidator {
	if importer := a.deps.Projects.Importer(); importer != nil {
		return importer.Validator
	}
	return nil
}

// previewSCL is POST /scl/preview: the file's IEDs, marking names already present. It
// imports nothing, so it also works while running (change 0186; Python refused it).
func (a *Area) previewSCL(w http.ResponseWriter, r *http.Request) error {
	source, _, err := upload(r, "file")
	if err != nil {
		return err
	}
	models, err := scl.Preview(source)
	if err != nil {
		return err
	}
	existing := map[string]bool{}
	ieds, _ := get(a.deps.Projects.Snapshot(), "ieds").([]any)
	for _, item := range ieds {
		if name, ok := get(object(item), "name").(string); ok && object(item) != nil {
			existing[name] = true
		}
	}
	list := make([]any, 0, len(models))
	for _, model := range models {
		if name, ok := get(model, "name").(string); ok && existing[name] {
			model.Set("unavailable_reason", "Already in this project")
		}
		list = append(list, model)
	}
	slots := maxImport - len(existing)
	if slots < 0 {
		slots = 0
	}
	writeJSON(w, 200, pyjson.NewObject().Set("ieds", list).Set("available_slots", slots))
	return nil
}

// importSCL is POST /scl/import with the previewed names in form field selected_names.
func (a *Area) importSCL(w http.ResponseWriter, r *http.Request) error {
	if err := a.requireStopped(); err != nil {
		return err
	}
	parsed, err := parseForm(r)
	if err != nil {
		return err
	}
	source, filename, err := parsed.file("file")
	if err != nil {
		return err
	}
	text, ok := parsed.values["selected_names"]
	if !ok {
		return apperr.New("Preview the file and choose which IEDs to import first")
	}
	selected, err := pyjson.Decode([]byte(text))
	if err != nil {
		return apperr.ValueError(pythonJSONError(text))
	}
	names, isList := selected.([]any)
	if !isList || len(names) < 1 || len(names) > maxImport {
		return apperr.New("Select between 1 and 20 IEDs")
	}
	if filename == "" {
		filename = "import.scd"
	}
	models, err := a.deps.Projects.AddSCL(source, filename, scl.Selection{Names: names})
	if err != nil {
		return err
	}
	writeJSON(w, 200, pyjson.NewObject().Set("ieds", modelIDs(models)))
	return nil
}

// template is POST /ieds/template {"name"}: add a laboratory template IED.
func (a *Area) template(w http.ResponseWriter, r *http.Request) error {
	if err := a.requireStopped(); err != nil {
		return err
	}
	raw, err := getJSON(r, false)
	if err != nil {
		return err
	}
	data := object(raw)
	if data == nil {
		return errNotDict
	}
	name := any("BayController")
	if value, ok := data.Get("name"); ok {
		name = value
	}
	text, ok := name.(string)
	if !ok {
		return errNotDict
	}
	models, err := a.deps.Projects.AddTemplate(text)
	if err != nil {
		return err
	}
	writeJSON(w, 200, pyjson.NewObject().Set("ieds", modelIDs(models)))
	return nil
}

func modelIDs(models []*pyjson.Object) []any {
	ids := make([]any, 0, len(models))
	for _, model := range models {
		ids = append(ids, get(model, "id"))
	}
	return ids
}

// source is GET /ieds/{ied_id}/source: the IED's original SCL file.
func (a *Area) source(w http.ResponseWriter, r *http.Request) error {
	ied, err := a.deps.Projects.FindIED(r.PathValue("ied_id"))
	if err != nil {
		return err
	}
	text, _ := get(ied, "source").(string)
	filename, _ := get(ied, "filename").(string)
	sendFile(w, r, []byte(text), "application/xml", filename)
	return nil
}

// exportSCL is POST /ieds/{ied_id}/export-scl {"overrides"}: the edited SCL document.
func (a *Area) exportSCL(w http.ResponseWriter, r *http.Request) error {
	raw, err := getJSON(r, false)
	if err != nil {
		return err
	}
	payload := object(raw)
	if payload == nil || !sameKeys(payload, "overrides") {
		return apperr.New("Provide an overrides object; engineering addresses remain unchanged")
	}
	ied, err := a.deps.Projects.FindIED(r.PathValue("ied_id"))
	if err != nil {
		return err
	}
	text, _ := get(ied, "source").(string)
	name, _ := get(ied, "name").(string)
	importer := a.deps.Projects.Importer()
	if importer == nil {
		importer = &scl.Importer{}
	}
	document, err := importer.Export(text, name, get(payload, "overrides"))
	if err != nil {
		return err
	}
	sendFile(w, r, document, "application/xml", name+".edited.scd")
	return nil
}
