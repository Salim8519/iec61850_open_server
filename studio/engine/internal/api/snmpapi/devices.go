package snmpapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/snmp/model"
)

func (a *Area) deviceRoutes() []api.Route {
	s := a.snmp
	return []api.Route{
		route("POST", "/devices", false, func(r *http.Request) (any, error) {
			var body struct {
				Profile string `json:"profile"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			device, err := s.AddDevice(body.Profile)
			return map[string]any{"device": device}, err
		}),
		route("PUT", "/devices/{id}", false, func(r *http.Request) (any, error) {
			var body struct {
				Device model.Device `json:"device"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			body.Device.ID = r.PathValue("id")
			device, err := s.UpdateDevice(body.Device, false)
			return map[string]any{"device": device}, err
		}),
		route("DELETE", "/devices/{id}", false, func(r *http.Request) (any, error) {
			return map[string]any{}, s.DeleteDevice(r.PathValue("id"))
		}),
		route("GET", "/devices/{id}/points", true, func(r *http.Request) (any, error) {
			points, live, err := s.Points(r.PathValue("id"))
			return map[string]any{"points": points, "live": live}, err
		}),
		route("PUT", "/devices/{id}/points", false, func(r *http.Request) (any, error) {
			var body struct {
				Points []model.Point `json:"points"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return map[string]any{}, s.SavePoints(r.PathValue("id"), body.Points)
		}),
		route("PUT", "/devices/{id}/value", false, func(r *http.Request) (any, error) {
			var body struct {
				OID   string `json:"oid"`
				Value string `json:"value"`
				Hex   bool   `json:"hex"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			point, err := s.SetValue(r.PathValue("id"), body.OID, body.Value, body.Hex)
			return map[string]any{"point": point}, err
		}),
		route("GET", "/devices/{id}/actions", false, func(r *http.Request) (any, error) {
			actions, err := s.Actions(r.PathValue("id"))
			return map[string]any{"actions": actions}, err
		}),
		route("POST", "/devices/{id}/trap", false, func(r *http.Request) (any, error) {
			var body struct {
				Action string `json:"action"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			results, err := s.Trigger(r.PathValue("id"), body.Action)
			return map[string]any{"results": results}, err
		}),
		route("GET", "/devices/{id}/points.csv", false, func(r *http.Request) (any, error) {
			text, name, err := s.ExportCSV(r.PathValue("id"))
			if err != nil {
				return nil, err
			}
			return &download{body: text, filename: name + ".csv"}, nil
		}),
		route("POST", "/import", false, func(r *http.Request) (any, error) {
			var body struct {
				Kind string `json:"kind"`
				Name string `json:"name"`
				Text string `json:"text"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.ImportDevice(body.Kind, body.Name, body.Text)
		}),
	}
}
