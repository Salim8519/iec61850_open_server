package toolsapi

import (
	"io"
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/testpanel"
	"studio-engine/internal/tools/trace"
)

func (a *Area) panelRoutes() []api.Route {
	return []api.Route{
		route("GET", "/project/test-panel", true, a.panel),
		route("PUT", "/project/test-panel", false, a.savePanel),
		route("POST", "/traces/preview", false, previewTrace),
	}
}

// panel lists the saved controls with each signal's current record and source state.
func (a *Area) panel(*http.Request) (any, error) {
	d := a.deps
	items := []any{}
	for _, raw := range d.Projects.TestPanel() {
		entry := raw.(*pyjson.Object)
		iedID, ref := value(entry, "ied_id", nil), value(entry, "ref", nil)
		signal, err := d.Runtime.Signal(iedID, ref)
		if err != nil {
			return nil, err
		}
		source := d.Runtime.Source(iedID, ref)
		overridden := any(false)
		automatic := false
		if source != nil {
			overridden = value(source, "overridden", false)
			automatic = source.Len() > 0 && value(source, "mode", nil) != "manual"
		}
		items = append(items, copyWith(copyWith(copyWith(entry, "signal", signal), "overridden", overridden),
			"automatic", automatic))
	}
	return pyjson.NewObject().Set("items", items), nil
}

// savePanel stores the saved controls; only while the simulation is stopped.
func (a *Area) savePanel(r *http.Request) (any, error) {
	d := a.deps
	err := d.Runtime.Stopped(func() error {
		entries, err := strictJSON(r)
		if err != nil {
			return err
		}
		clean, err := testpanel.Validate(entries, d.Projects.IEDs())
		if err != nil {
			return err
		}
		return d.Projects.SaveTestPanel(clean)
	})
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("ok", true), nil
}

// previewTrace reads an uploaded CSV trace and shows its columns and first rows.
func previewTrace(r *http.Request) (any, error) {
	if r.ContentLength > maxBody {
		return nil, &httpError{http.StatusRequestEntityTooLarge, "The data value transmitted exceeds the capacity limit."}
	}
	refused := &httpError{http.StatusBadRequest, badRequest}
	if err := r.ParseMultipartForm(32 << 20); err != nil || r.MultipartForm == nil {
		return nil, refused
	}
	files := r.MultipartForm.File["file"]
	if len(files) == 0 {
		return nil, refused
	}
	file, err := files[0].Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()
	payload, err := io.ReadAll(io.LimitReader(file, trace.MaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > trace.MaxBytes {
		return nil, apperr.ValueError("CSV trace exceeds 5 MB")
	}
	text, err := trace.DecodeUTF8Sig(payload)
	if err != nil {
		return nil, err
	}
	columns, rows, err := trace.ReadRows(text)
	if err != nil {
		return nil, err
	}
	first := []any{}
	for index, row := range rows {
		if index == 5 {
			break
		}
		first = append(first, toStrings(row))
	}
	return pyjson.NewObject().Set("columns", toStrings(columns)).Set("rows", first).
		Set("row_count", int64(len(rows))), nil
}
