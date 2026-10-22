package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// maxUnticked is how many unticked keys are stored.
const maxUnticked = 20000

func (a *Area) masterPointRoutes() []api.Route {
	m := a.deps.Master
	return []api.Route{
		route("GET", "/tools/master/points", true, func(*http.Request) (any, error) {
			return wrapPoints(m.Points())
		}),
		route("PUT", "/tools/master/selection", false, a.saveSelection),
		route("PUT", "/tools/master/label", false, a.saveLabel),
		route("GET", "/tools/master/points.csv", false, func(*http.Request) (any, error) {
			protocol := pytext.Str(value(m.Config(), "protocol", nil))
			rows, err := m.Points()
			if err != nil {
				return nil, err
			}
			return &download{body: gateway.ExportMasterCSV(rows), contentType: "text/csv; charset=utf-8",
				disposition: "attachment; filename=master-" + protocol + "-points.csv"}, nil
		}),
		route("POST", "/tools/master/labels.csv", false, a.importLabels),
		route("PUT", "/tools/master/ranges", false, func(r *http.Request) (any, error) {
			current := m.Config()
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			ranges, err := gateway.ValidateRanges(value(data, "modbus_ranges", nil))
			if err != nil {
				return nil, err
			}
			if err := m.Save(copyWith(current, "modbus_ranges", ranges)); err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("modbus_ranges", ranges), nil
		}),
		route("PUT", "/tools/master/routes", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			routes, err := gateway.ValidateRoutes(value(data, "routes", nil))
			if err != nil {
				return nil, err
			}
			if err := m.Save(copyWith(m.Config(), "routes", routes)); err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("routes", routes), nil
		}),
		route("POST", "/tools/master/probe", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			table, start, end, err := gateway.ValidateProbe(value(data, "table", nil), value(data, "start", nil),
				value(data, "end", nil))
			if err != nil {
				return nil, err
			}
			return m.FindRegisters(table, start, end)
		}),
		route("POST", "/tools/master/probe/stop", false, func(*http.Request) (any, error) {
			return m.StopFindRegisters()
		}),
		route("POST", "/tools/master/probe/ranges", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			found, _ := value(m.Probe.Snapshot(), "found", []any{}).([]any)
			current := m.Config()
			added, err := gateway.RangesFromProbe(found, value(data, "data_type", "uint16"))
			if err != nil {
				return nil, err
			}
			existing, _ := value(current, "modbus_ranges", nil).([]any)
			ranges, err := gateway.ValidateRanges(append(append([]any{}, existing...), added...))
			if err != nil {
				return nil, err
			}
			if err := m.Save(copyWith(current, "modbus_ranges", ranges)); err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("modbus_ranges", ranges), nil
		}),
	}
}

func (a *Area) saveSelection(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	unticked, ok := value(data, "unticked", nil).([]any)
	for _, key := range unticked {
		if text, isText := key.(string); !isText || pytext.Len(text) > 40 {
			ok = false
		}
	}
	if !ok {
		return nil, apperr.ValueError("Send the unticked point keys as a list")
	}
	if len(unticked) > maxUnticked {
		unticked = unticked[:maxUnticked]
	}
	m := a.deps.Master
	current := m.Config()
	protocol := pytext.Str(value(current, "protocol", nil))
	stored, _ := value(current, "unticked", nil).(*pyjson.Object)
	if err := m.Save(copyWith(current, "unticked", copyWith(orObject(stored), protocol, unticked))); err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("unticked", unticked), nil
}
