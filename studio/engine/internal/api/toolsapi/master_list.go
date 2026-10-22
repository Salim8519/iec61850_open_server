package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/gateway/masterpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// MasterGoOnlyRoutes are the SCADA master routes only the Go engine serves: the repeat sender
// (change 0169), the master's own point list (ADR-0110) and Stop for Find registers (0178). The frozen Python engine answers
// them 404; the Flask route comparison allows them.
var MasterGoOnlyRoutes = map[string]bool{"GET /api/v1/tools/master/cyclic": true,
	"POST /api/v1/tools/master/cyclic": true, "POST /api/v1/tools/master/cyclic/stop": true,
	"GET /api/v1/tools/master/list": true, "PUT /api/v1/tools/master/list": true,
	"PUT /api/v1/tools/master/list/check-ied": true, "POST /api/v1/tools/master/list/new": true,
	"POST /api/v1/tools/master/list/rows": true, "GET /api/v1/tools/master/list.xlsx": true,
	"POST /api/v1/tools/master/list.xlsx": true, "GET /api/v1/tools/master/list.csv": true,
	"POST /api/v1/tools/master/list/copy-gateway": true, "POST /api/v1/tools/master/list/from-received": true,
	"GET /api/v1/tools/master/list/compare": true, "POST /api/v1/tools/master/probe/stop": true}

// masterListRoutes plan and save the master's own point list. Planning routes never change the
// list: the page adds the planned points and saves the whole list with PUT (save always,
// ADR-0101), which checks it again.
func (a *Area) masterListRoutes() []api.Route {
	m := a.deps.Master
	return append([]api.Route{
		route("GET", "/tools/master/list", false, func(*http.Request) (any, error) { return a.masterList(), nil }),
		route("PUT", "/tools/master/list", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			if _, found := data.Get("points"); !found {
				return nil, apperr.ValueError("Send the point list as points")
			}
			if _, err := m.SaveList(value(data, "points", nil)); err != nil {
				return nil, err
			}
			return a.masterList(), nil
		}),
		route("PUT", "/tools/master/list/check-ied", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			on, isBool := value(data, "on", nil).(bool)
			if !isBool {
				return nil, apperr.ValueError("Send on: true or false")
			}
			if err := m.SetCheckIED(on); err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("check_ied", on), nil
		}),
		route("POST", "/tools/master/list/new", false, a.masterListNew),
		route("POST", "/tools/master/list/copy-gateway", false, func(r *http.Request) (any, error) {
			if _, err := body(r); err != nil { // read it all: an unread body can reset the large answer
				return nil, err
			}
			protocol, current := m.List()
			lists, _ := value(a.deps.Gateway.Config(), "points", nil).(*pyjson.Object)
			source, _ := value(lists, protocol, []any{}).([]any)
			names := map[string]string{}
			for _, ied := range a.deps.Projects.IEDs() {
				id, _ := value(ied, "id", "").(string)
				name, _ := value(ied, "name", "").(string)
				names[id] = name
			}
			results, notes := masterpoints.FromGateway(protocol, source, current, names)
			return masterpoints.PlanJSON(results).Set("notes", toStrings(notes)).
				Set("gateway_points", int64(len(source))), nil
		}),
		route("POST", "/tools/master/list/from-received", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			keys := []string{}
			for _, key := range listOf(value(data, "keys", nil)) {
				if text, isText := key.(string); isText {
					keys = append(keys, text)
				}
			}
			rows, err := m.Points()
			if err != nil {
				return nil, err
			}
			protocol, current := m.List()
			return masterpoints.PlanJSON(masterpoints.FromReceived(protocol, rows, keys, current)), nil
		}),
		route("GET", "/tools/master/list/compare", false, func(*http.Request) (any, error) { return a.compare() }),
	}, a.masterListFileRoutes()...)
}

// masterList is the list answer: the protocol, its points, its type list, the switch of the
// Check against IED column and the Modbus reads the list adds to every poll.
func (a *Area) masterList() *pyjson.Object {
	m := a.deps.Master
	protocol, list := m.List()
	types := []any{}
	for _, entry := range addpoints.MasterTypes(protocol) {
		types = append(types, masterTypeJSON(entry))
	}
	auto := []any{}
	if protocol == "modbus" {
		auto = masterpoints.PollRanges(list)
	}
	return pyjson.NewObject().Set("protocol", protocol).Set("points", list).Set("types", types).
		Set("check_ied", m.CheckIED()).Set("auto_ranges", auto)
}

func masterTypeJSON(entry addpoints.MasterType) *pyjson.Object {
	result := pyjson.NewObject().Set("kind", entry.Key).Set("name", entry.Name).Set("type_id", entry.TypeID).
		Set("most_used", entry.MostUsed).Set("group", entry.Group).Set("command", entry.Command)
	if entry.Table != "" {
		result.Set("table", entry.Table).Set("data_type", entry.DataType).Set("registers", entry.Registers())
	}
	return result
}

// masterListNew plans one point typed in the form, or a range with count and from.
func (a *Area) masterListNew(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	protocol, current := a.deps.Master.List()
	if count, isRange := value(data, "count", nil).(int64); isRange {
		from, isInt := value(data, "from", nil).(int64)
		if !isInt {
			return nil, apperr.ValueError("Type the address of the first point")
		}
		typed := pytext.Str(value(data, "type", ""))
		name := pytext.Head(pytext.Strip(pytext.Str(value(data, "name", ""))), 100)
		return masterpoints.PlanJSON(masterpoints.Range(protocol, typed, from, int(count), name, current)), nil
	}
	return masterpoints.PlanJSON(masterpoints.One(protocol, data, current)), nil
}

// compare is expected vs received for the page, with the optional Check against IED column:
// for each point copied from the gateway, the gateway service says what its mapping holds now.
func (a *Area) compare() (any, error) {
	m := a.deps.Master
	rows, err := m.Points()
	if err != nil {
		return nil, err
	}
	protocol, list := m.List()
	answer := masterpoints.Compare(protocol, list, rows, m.Link())
	check := m.CheckIED()
	answer.Set("check_ied", check)
	if !check {
		return answer, nil
	}
	items, _ := value(answer, "rows", nil).([]any)
	for _, raw := range items {
		item := raw.(*pyjson.Object)
		point, _ := value(item, "point", nil).(*pyjson.Object)
		from, address, kind, copied := masterpoints.FromOf(point)
		if !copied {
			continue
		}
		item.Set("check", masterpoints.CheckAgainstIED(protocol, item, a.deps.Gateway.Mapped(from, address, kind)))
	}
	return answer, nil
}

func listOf(raw any) []any {
	items, _ := raw.([]any)
	return items
}
