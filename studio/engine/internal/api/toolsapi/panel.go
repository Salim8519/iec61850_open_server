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
		// GET /project/test-panel is runtimeapi's (same Python handler, real coordinator).
		route("PUT", "/project/test-panel", false, a.savePanel),
		route("POST", "/traces/preview", false, previewTrace),
	}
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
