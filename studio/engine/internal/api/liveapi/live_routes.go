package liveapi

import (
	"errors"
	"net/http"
	"strings"

	"studio-engine/internal/acquisition/health"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

var errUnhashable = errors.New("unhashable type")

// live is GET /live: {"goose": snapshot, "sessions": [status with comparison/health None]}.
func (s *Service) live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Live.Status())
}

func (s *Service) comparisonOf(key string) (*pyjson.Object, error) {
	session, err := s.Live.Session(key)
	if err != nil {
		return nil, err
	}
	comparison := pyv.GetObj(session.Status(), "comparison")
	if comparison == nil {
		return nil, apperr.ValueError("The comparison is ready after ScadaBench has connected to the device and read its model")
	}
	return comparison, nil
}

func (s *Service) comparison(w http.ResponseWriter, r *http.Request) {
	comparison, err := s.comparisonOf(r.PathValue("key"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, comparison)
}

// comparisonCSV writes csv.writer rows (CRLF, minimal quoting) of the SCD-vs-device rows.
func (s *Service) comparisonCSV(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	comparison, err := s.comparisonOf(key)
	if err != nil {
		writeError(w, err)
		return
	}
	var out strings.Builder
	out.WriteString("reference,fc,result,scd_type,device_type\r\n")
	for _, row := range pyv.Objects(pyv.GetList(comparison, "signals")) {
		fields := []any{pyv.Get(row, "ref"), pyv.Get(row, "fc"), pyv.Get(row, "category"),
			orEmpty(pyv.Get(row, "scd_type")), orEmpty(pyv.Get(row, "device_type"))}
		for index, field := range fields {
			if index > 0 {
				out.WriteByte(',')
			}
			if field != nil {
				out.WriteString(health.CSVField(pyv.PyStr(field)))
			}
		}
		out.WriteString("\r\n")
	}
	ied, err := s.Deps.Projects.FindIED(key)
	if err != nil {
		writeError(w, err)
		return
	}
	writeDownload(w, "text/csv", "attachment; filename="+pyv.PyStr(pyv.Get(ied, "name"))+"-scd-vs-device.csv", out.String())
}

func orEmpty(value any) any {
	if !pyv.Truthy(value) {
		return ""
	}
	return value
}

// read is POST /live/<key>/read: read 1..500 known references now.
func (s *Service) read(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	request, err := body(r)
	if err != nil {
		writeError(w, err)
		return
	}
	refs, ok := pyv.Get(request, "refs").([]any)
	if !ok || len(refs) == 0 || len(refs) > 500 {
		writeError(w, apperr.ValueError("Send between 1 and 500 references to read"))
		return
	}
	ied, err := s.Deps.Projects.FindIED(key)
	if err != nil {
		writeError(w, err)
		return
	}
	index := s.signals.For(ied)
	items := []any{}
	for _, ref := range refs {
		switch value := ref.(type) {
		case []any, *pyjson.Object:
			writeError(w, errUnhashable)
			return
		case string:
			if signal := index.Signal(value); signal != nil {
				items = append(items, pyv.O("ref", value, "fc", pyv.Get(signal, "fc")))
			}
		}
	}
	session, err := s.Live.Session(key)
	if err != nil {
		writeError(w, err)
		return
	}
	result, err := session.Read(items, "live_poll")
	if err != nil {
		writeError(w, err)
		return
	}
	values := pyjson.NewObject()
	if s.Deps.Values.Has(key) {
		for _, ref := range refs {
			text, _ := ref.(string)
			if value, state, found := s.Deps.Values.Get(key, text); found {
				values.Set(text, pyv.O("value", value, "live_state", state))
			}
		}
	}
	writeJSON(w, http.StatusOK, pyv.O("values", values, "errors", pyv.Get(result, "errors")))
}
