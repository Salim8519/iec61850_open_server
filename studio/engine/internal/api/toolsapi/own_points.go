package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/gateway/addpoints"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// OwnPointRoutes are the Go-only routes of the gateway's own points (lane G) and of Add points
// (ADR-0105, add_points.go); the frozen Python engine has neither and answers them 404.
var OwnPointRoutes = map[string]bool{"POST /api/v1/tools/gateway/points/new": true,
	"PUT /api/v1/tools/gateway/points/value": true, "GET /api/v1/tools/gateway/types": true,
	"POST /api/v1/tools/gateway/candidates": true, "POST /api/v1/tools/gateway/points/from-ied": true,
	"POST /api/v1/tools/gateway/points/rows": true, "GET /api/v1/tools/gateway/points.xlsx": true,
	"POST /api/v1/tools/gateway/points.xlsx": true}

func (a *Area) ownRoutes() []api.Route {
	d := a.deps
	return []api.Route{
		// A new own point for the page's draft list: next own id and next free address.
		route("POST", "/tools/gateway/points/new", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			protocol, err := gatewayProtocol(value(data, "protocol", nil))
			if err != nil {
				return nil, err
			}
			kind, _ := value(data, "kind", nil).(string)
			if count, isRange := value(data, "count", nil).(int64); isRange { // "Add a range" (ADR-0105)
				from, _ := value(data, "from", nil).(int64)
				points, _ := value(data, "points", []any{}).([]any)
				name := pytext.Str(value(data, "name", ""))
				return rowsAnswer(addpoints.OwnRange(protocol, kind, int(count), from, name, points), nil), nil
			}
			point, err := gateway.NewOwnPoint(protocol, kind, value(data, "points", []any{}))
			if err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("point", point), nil
		}),
		// The value of a saved own point; while the gateway runs SCADA gets it at once.
		route("PUT", "/tools/gateway/points/value", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			id, isText := value(data, "point_id", nil).(string)
			raw, found := data.Get("value")
			if !isText || !found {
				return nil, apperr.ValueError("Send point_id and value")
			}
			row, err := d.Gateway.SetOwnValue(id, raw)
			if err != nil {
				return nil, err
			}
			return pyjson.NewObject().Set("point", row), nil
		}),
	}
}
