package toolsapi

import (
	"encoding/base64"
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/gateway/masterpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

const xlsxType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

// masterListFileRoutes read pasted rows, CSV and the engine-made Excel template of the master
// point list, and export the list as CSV. Reading only plans; the page saves the list.
func (a *Area) masterListFileRoutes() []api.Route {
	m := a.deps.Master
	return []api.Route{
		route("POST", "/tools/master/list/rows", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			protocol, current := m.List()
			results, columns := masterpoints.PlanText(protocol, pytext.Str(value(data, "text", "")), current)
			return withColumns(masterpoints.PlanJSON(results), columns), nil
		}),
		route("GET", "/tools/master/list.xlsx", false, func(*http.Request) (any, error) {
			protocol, _ := m.List()
			book, err := masterpoints.Template(protocol)
			if err != nil {
				return nil, err
			}
			return &download{body: string(book), contentType: xlsxType,
				disposition: "attachment; filename=master-" + protocol + "-template.xlsx"}, nil
		}),
		route("POST", "/tools/master/list.xlsx", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			raw, err := base64.StdEncoding.DecodeString(pytext.Str(value(data, "data", "")))
			if err != nil {
				return nil, apperr.ValueError(addpoints.ErrNotWorkbook.Error())
			}
			protocol, current := m.List()
			results, columns, err := masterpoints.PlanWorkbook(protocol, raw, current)
			if err != nil {
				return nil, apperr.ValueError(err.Error())
			}
			return withColumns(masterpoints.PlanJSON(results), columns), nil
		}),
		route("GET", "/tools/master/list.csv", false, func(*http.Request) (any, error) {
			protocol, list := m.List()
			return &download{body: string(masterpoints.ExportCSV(protocol, list)), contentType: "text/csv; charset=utf-8",
				disposition: "attachment; filename=master-" + protocol + "-point-list.csv"}, nil
		}),
	}
}

func withColumns(plan *pyjson.Object, columns []string) *pyjson.Object {
	return plan.Set("columns", toStrings(columns))
}
