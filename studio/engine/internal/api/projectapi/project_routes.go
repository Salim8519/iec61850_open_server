package projectapi

import (
	"math/big"
	"net/http"
	"unicode/utf8"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/projectfile"
)

// getProject is GET /project: snapshot + runtime, running_ieds, startup_recovery.
func (a *Area) getProject(w http.ResponseWriter, r *http.Request) error {
	body := a.deps.Projects.Snapshot()
	state, running := a.runtimeState()
	var recovery any
	if a.deps.Startup != nil {
		if result, _ := a.deps.Startup.Result(); result != nil {
			recovery = result
		}
	}
	body.Set("runtime", state).Set("running_ieds", running).Set("startup_recovery", recovery)
	writeJSON(w, 200, body)
	return nil
}

// putDiagram is PUT /project/diagram.
func (a *Area) putDiagram(w http.ResponseWriter, r *http.Request) error {
	if err := a.requireStopped(); err != nil {
		return err
	}
	raw, _ := getJSON(r, true)
	data := object(raw)
	if data == nil && truthy(raw) {
		return diagramPayloadError(raw)
	}
	if data == nil {
		data = pyjson.NewObject()
	}
	if !sameKeys(data, "diagram", "revision", "tab_id") {
		return apperr.ValueError("Diagram update requires its project revision and tab identity")
	}
	revision, ok := pythonInt(get(data, "revision"))
	tab, isText := get(data, "tab_id").(string)
	if !ok || !isText || utf8.RuneCountInString(tab) < 1 || utf8.RuneCountInString(tab) > 64 {
		return apperr.ValueError("Invalid project revision or tab identity")
	}
	if err := a.deps.Projects.Diagram(get(data, "diagram"), &revision, tab); err != nil {
		return err
	}
	writeJSON(w, 200, pyjson.NewObject().Set("ok", true).Set("revision", a.deps.Projects.Revision()))
	return nil
}

// diagramPayloadError is what `set(data) != {...}` does with truthy non-dict JSON: a
// string's characters never match (ValueError); a list matching the key set, holding
// unhashable items, or a number raise TypeError in Python (the generic 500).
func diagramPayloadError(raw any) error {
	keyError := apperr.ValueError("Diagram update requires its project revision and tab identity")
	switch v := raw.(type) {
	case string:
		return keyError
	case []any:
		seen, other := map[string]bool{}, false
		for _, item := range v {
			switch key := item.(type) {
			case *pyjson.Object, []any:
				return errNotDict
			case string:
				seen[key] = true
			default:
				other = true
			}
		}
		if !other && len(seen) == 3 && seen["diagram"] && seen["revision"] && seen["tab_id"] {
			return errNotDict
		}
		return keyError
	}
	return errNotDict
}

// pythonInt accepts what isinstance(value, int) accepts (bool included) as an int64;
// integers beyond int64 can never equal a revision and map to -1.
func pythonInt(value any) (int64, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case *big.Int:
		return -1, true
	}
	return 0, false
}

func get(o *pyjson.Object, key string) any {
	value, _ := o.Get(key)
	return value
}

func sameKeys(o *pyjson.Object, keys ...string) bool {
	if o.Len() != len(keys) {
		return false
	}
	for _, key := range keys {
		if _, ok := o.Get(key); !ok {
			return false
		}
	}
	return true
}

// saveProject is POST /project/save.
func (a *Area) saveProject(w http.ResponseWriter, r *http.Request) error {
	if err := a.deps.Projects.Save(); err != nil {
		return err
	}
	writeJSON(w, 200, pyjson.NewObject().Set("ok", true))
	return nil
}

// downloadProject is GET /project/download.
func (a *Area) downloadProject(w http.ResponseWriter, r *http.Request) error {
	payload, err := a.deps.Projects.Export()
	if err != nil {
		return err
	}
	sendFile(w, r, payload, "application/zip", projectfile.DownloadName)
	return nil
}

// openProject is POST /project/open: restore an uploaded archive and answer the snapshot.
func (a *Area) openProject(w http.ResponseWriter, r *http.Request) error {
	if err := a.requireStopped(); err != nil {
		return err
	}
	payload, name, err := upload(r, "file")
	if err != nil {
		return err
	}
	if projectfile.IsOldName(name) { // a .subsim file is refused in plain words (ADR-0096)
		return apperr.Coded(projectfile.OldProjectMessage, projectfile.OldProjectCode)
	}
	if err := a.deps.Projects.Restore(payload, true); err != nil {
		return err
	}
	writeJSON(w, 200, a.deps.Projects.Snapshot())
	return nil
}
