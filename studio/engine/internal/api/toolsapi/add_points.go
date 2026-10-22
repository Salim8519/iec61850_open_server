package toolsapi

import (
	"encoding/base64"
	"net/http"
	"strings"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// The Go-only "Add points" routes (ADR-0105). They only plan and check: nothing is added
// until the page saves the list with PUT /tools/gateway/points, which checks it again.
func (a *Area) addRoutes() []api.Route {
	d := a.deps
	return []api.Route{
		route("GET", "/tools/gateway/types", false, func(r *http.Request) (any, error) {
			protocol, err := protocolQuery(r)
			if err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("types", typesJSON(addpoints.Types(protocol))), nil
		}),
		route("POST", "/tools/gateway/candidates", false, a.candidates),
		route("POST", "/tools/gateway/points/from-ied", false, a.fromIED),
		route("POST", "/tools/gateway/points/rows", false, func(r *http.Request) (any, error) {
			data, protocol, err := gatewayBody(r)
			if err != nil {
				return nil, err
			}
			return a.checkCells(protocol, addpoints.SplitText(pytext.Str(value(data, "text", ""))), data, false)
		}),
		route("GET", "/tools/gateway/points.xlsx", false, func(r *http.Request) (any, error) {
			protocol, err := protocolQuery(r)
			if err != nil {
				return nil, err
			}
			book, err := addpoints.Template(protocol, addpoints.NewIndex(d.Projects.IEDs()))
			if err != nil {
				return nil, err
			}
			return &download{body: string(book), disposition: "attachment; filename=gateway-" + protocol + "-template.xlsx",
				contentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}, nil
		}),
		route("POST", "/tools/gateway/points.xlsx", false, func(r *http.Request) (any, error) {
			data, protocol, err := gatewayBody(r)
			if err != nil {
				return nil, err
			}
			raw, err := base64.StdEncoding.DecodeString(pytext.Str(value(data, "data", "")))
			if err != nil {
				return nil, apperr.ValueError(addpoints.ErrNotWorkbook.Error())
			}
			cells, err := addpoints.ReadWorkbook(raw)
			if err != nil {
				return nil, apperr.ValueError(err.Error())
			}
			return a.checkCells(protocol, cells, data, true)
		}),
	}
}

// gatewayBody is the JSON body and its checked "protocol".
func gatewayBody(r *http.Request) (*pyjson.Object, string, error) {
	data, err := body(r)
	if err != nil {
		return nil, "", err
	}
	protocol, err := gatewayProtocol(value(data, "protocol", nil))
	return data, protocol, err
}

func pointList(data *pyjson.Object) []any {
	points, _ := value(data, "points", []any{}).([]any)
	return points
}

func (a *Area) candidates(r *http.Request) (any, error) {
	data, _, err := gatewayBody(r)
	if err != nil {
		return nil, err
	}
	kind, _ := value(data, "kind", nil).(string)
	if !gateway.KnownKind(kind) {
		return nil, apperr.ValueError("Choose a point type")
	}
	ied := addpoints.NewIndex(a.deps.Projects.IEDs()).ByID(pytext.Str(value(data, "ied", "")))
	if ied == nil {
		return nil, apperr.ValueError("This IED is not in the project")
	}
	items, total := addpoints.Candidates(ied, kind, pytext.Str(value(data, "q", "")), addpoints.TakenIDs(pointList(data)))
	list := make([]any, 0, len(items))
	for _, item := range items {
		row := pyjson.NewObject().Set("ref", item.Ref).Set("name", item.Name).Set("description", item.Description).
			Set("value_kind", item.ValueKind).Set("fc", item.FC).Set("taken", item.Taken).Set("suggested", item.Suggested)
		if item.CDC != "" {
			row.Set("cdc", item.CDC).Set("status_ref", item.StatusRef)
		}
		list = append(list, row)
	}
	return pyjson.NewObject().Set("items", list).Set("total", int64(total)).Set("ied", iedJSON(ied)), nil
}

func (a *Area) fromIED(r *http.Request) (any, error) {
	data, protocol, err := gatewayBody(r)
	if err != nil {
		return nil, err
	}
	ieds := a.deps.Projects.IEDs()
	ied := addpoints.NewIndex(ieds).ByID(pytext.Str(value(data, "ied", "")))
	if ied == nil {
		return nil, apperr.ValueError("This IED is not in the project")
	}
	kind, _ := value(data, "kind", nil).(string)
	raw, _ := value(data, "picks", []any{}).([]any)
	current := pointList(data)
	if len(raw) == 0 || len(current)+len(raw) > gateway.MaxPoints {
		return nil, apperr.ValueError("Tick at least one signal (a gateway holds at most 5000 points)")
	}
	picks := make([]addpoints.Pick, 0, len(raw))
	for _, item := range raw {
		pick, _ := item.(*pyjson.Object)
		if pick == nil {
			return nil, apperr.ValueError("Send each ticked signal as an object")
		}
		address, typed := value(pick, "address", nil).(int64)
		picks = append(picks, addpoints.Pick{Ref: pytext.Str(value(pick, "ref", "")),
			Name: strings.TrimSpace(pytext.Str(value(pick, "name", ""))), Address: address, HasAddress: typed})
	}
	withStatus, _ := value(data, "with_status", true).(bool)
	results := addpoints.FromIED(protocol, ied, kind, picks, withStatus, current, ieds)
	return rowsAnswer(results, nil).Set("ied", iedJSON(ied)), nil
}

// checkCells checks pasted or uploaded (file) rows; points are the new points, only when
// every row can be added.
func (a *Area) checkCells(protocol string, cells [][]string, data *pyjson.Object, file bool) (any, error) {
	ieds := a.deps.Projects.IEDs()
	index := addpoints.NewIndex(ieds)
	rows, numbers, columns := addpoints.ReadTable(cells, protocol, index)
	current := pointList(data)
	if len(rows) == 0 && file {
		return nil, apperr.ValueError("No rows were found in the file. Fill in the rows " +
			"(below the title row) of the Points sheet, save the file and choose it again.")
	}
	if len(rows) == 0 {
		return nil, apperr.ValueError("No rows were found. Copy the rows in Excel (with or without the title row) " +
			"and paste them here.")
	}
	if len(current)+len(rows) > gateway.MaxPoints {
		return nil, apperr.ValueError("A gateway holds at most 5000 points; these rows would make more")
	}
	results := addpoints.CheckRows(protocol, rows, numbers, index, ieds, current)
	return rowsAnswer(results, columns), nil
}

// rowsAnswer is {rows, points, ok[, columns]}: points only when every row can be added.
func rowsAnswer(results []addpoints.Result, columns []string) *pyjson.Object {
	ok := addpoints.OK(results)
	points := []any{}
	if ok {
		points = addpoints.Points(results)
	}
	answer := pyjson.NewObject().Set("rows", addpoints.JSON(results)).Set("points", points).Set("ok", ok)
	if columns != nil {
		answer.Set("columns", toStrings(columns))
	}
	return answer
}

func iedJSON(ied *addpoints.IED) *pyjson.Object {
	return pyjson.NewObject().Set("id", ied.ID).Set("name", ied.Name).Set("role", ied.Role)
}

func typesJSON(entries []addpoints.TypeEntry) []any {
	list := make([]any, 0, len(entries))
	for _, entry := range entries {
		list = append(list, pyjson.NewObject().Set("kind", entry.Kind).Set("name", entry.Name).
			Set("type_id", entry.TypeID).Set("most_used", entry.MostUsed))
	}
	return list
}
