package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

func protocolQuery(r *http.Request) (string, error) {
	text, found := query(r, "protocol")
	if !found {
		return gatewayProtocol(nil)
	}
	return gatewayProtocol(text)
}

func toStrings(values []string) []any {
	result := make([]any, len(values))
	for index, item := range values {
		result[index] = item
	}
	return result
}

// named adds the IED name and monitor_only (role is not "simulated") to proposed points.
func (a *Area) named(points []any) []any {
	names, roles := map[string]any{}, map[string]any{}
	for _, ied := range a.deps.Projects.IEDs() {
		id := pytext.Str(value(ied, "id", nil))
		names[id], roles[id] = value(ied, "name", nil), value(ied, "role", nil)
	}
	result := make([]any, 0, len(points))
	for _, raw := range points {
		point := raw.(*pyjson.Object)
		if gateway.IsOwn(point) { // Go-only: an own point has no IED and always takes commands
			result = append(result, copyWith(copyWith(point, "ied_name", gateway.OwnName), "monitor_only", false))
			continue
		}
		key := pytext.Str(value(point, "ied", nil))
		name, found := names[key]
		if !found {
			name = "?"
		}
		result = append(result, copyWith(copyWith(point, "ied_name", name), "monitor_only", roles[key] != "simulated"))
	}
	return result
}

func (a *Area) pointRoutes() []api.Route {
	d := a.deps
	return []api.Route{
		route("GET", "/tools/gateway/points", true, func(r *http.Request) (any, error) {
			protocol, err := protocolQuery(r)
			if err != nil {
				return nil, err
			}
			return wrapPoints(d.Gateway.Table(protocol))
		}),
		route("PUT", "/tools/gateway/points", false, a.savePoints),
		route("POST", "/tools/gateway/propose", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			protocol, err := gatewayProtocol(value(data, "protocol", nil))
			if err != nil {
				return nil, err
			}
			proposal, notes, err := gateway.Propose(d.Projects.IEDs(), protocol)
			if err != nil {
				return nil, err
			}
			if keep, found := data.Get("keep"); found { // Go-only: the page's own points stay
				var moved []string
				proposal, moved = gateway.KeepOwn(protocol, proposal, keep)
				notes = append(notes, moved...)
			}
			return pyjson.NewObject().Set("points", a.named(proposal)).Set("notes", toStrings(notes)), nil
		}),
		route("GET", "/tools/gateway/points.csv", false, func(r *http.Request) (any, error) {
			protocol, err := protocolQuery(r)
			if err != nil {
				return nil, err
			}
			saved, _ := value(d.Gateway.Config(), "points", nil).(*pyjson.Object)
			points, _ := value(orObject(saved), protocol, []any{}).([]any)
			text, err := gateway.ExportCSV(points, d.Projects.IEDs())
			if err != nil {
				return nil, err
			}
			return &download{body: text, contentType: "text/csv; charset=utf-8",
				disposition: "attachment; filename=gateway-" + protocol + "-points.csv"}, nil
		}),
		route("POST", "/tools/gateway/points.csv", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			protocol, err := gatewayProtocol(value(data, "protocol", nil))
			if err != nil {
				return nil, err
			}
			ieds := d.Projects.IEDs()
			parsed, problems, err := gateway.ImportCSV(pytext.Str(value(data, "text", "")), ieds)
			if err != nil {
				return nil, err
			}
			clean, invalid, err := gateway.ValidatePoints(protocol, parsed, ieds)
			if err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("points", a.named(clean)).
				Set("problems", toStrings(append(problems, invalid...))), nil
		}),
	}
}

// wrapPoints is jsonify(points=...) around a value that may have failed.
func wrapPoints(points []any, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	return pyjson.NewObject().Set("points", points), nil
}

func orObject(object *pyjson.Object) *pyjson.Object {
	if object == nil {
		return pyjson.NewObject()
	}
	return object
}

func (a *Area) savePoints(r *http.Request) (any, error) {
	data, err := body(r)
	if err != nil {
		return nil, err
	}
	d := a.deps
	if d.Gateway.State() == "running" {
		return nil, apperr.ValueError("Stop the gateway before changing its point list")
	}
	protocol, err := gatewayProtocol(value(data, "protocol", nil))
	if err != nil {
		return nil, err
	}
	clean, problems, err := gateway.ValidatePoints(protocol, value(data, "points", nil), d.Projects.IEDs())
	if err != nil {
		return nil, err
	}
	if len(problems) > 0 {
		return &answer{status: http.StatusBadRequest, body: pyjson.NewObject().
			Set("error", "Some points need attention before they can be saved").Set("problems", toStrings(problems))}, nil
	}
	current := d.Gateway.Config()
	saved, _ := value(current, "points", nil).(*pyjson.Object)
	if err := d.Projects.SaveGateway(copyWith(current, "points", copyWith(orObject(saved), protocol, clean))); err != nil {
		return nil, err
	}
	return wrapPoints(d.Gateway.Table(protocol))
}
