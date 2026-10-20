package snmpapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/service"
)

func (a *Area) readerRoutes() []api.Route {
	s := a.snmp
	return []api.Route{
		route("POST", "/read/connect", false, func(r *http.Request) (any, error) {
			var body struct {
				Target model.Target `json:"target"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.Connect(body.Target)
		}),
		route("POST", "/read/disconnect", false, func(*http.Request) (any, error) { return s.Disconnect(), nil }),
		route("POST", "/read/walk", false, func(r *http.Request) (any, error) {
			var body struct {
				Root string `json:"root"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.Walk(body.Root)
		}),
		route("GET", "/read/values", true, func(*http.Request) (any, error) {
			return map[string]any{"values": s.Values()}, nil
		}),
		route("PUT", "/read/poll", false, func(r *http.Request) (any, error) {
			var body struct {
				EveryMS int `json:"every_ms"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.SetPoll(body.EveryMS)
		}),
		route("PUT", "/read/watch", false, func(r *http.Request) (any, error) {
			var body struct {
				OIDs []string `json:"oids"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.Watch(body.OIDs)
		}),
		route("POST", "/read/set", false, func(r *http.Request) (any, error) {
			var body struct {
				Point   model.Point `json:"point"`
				Confirm bool        `json:"confirm"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			if !body.Confirm {
				return nil, apperr.ValueError("Confirm the write first: it changes the real device")
			}
			values, err := s.SetDevice(body.Point)
			return map[string]any{"values": values}, err
		}),
		route("POST", "/read/save", false, func(r *http.Request) (any, error) {
			var body struct {
				Name string `json:"name"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.SaveReadDevice(body.Name)
		}),
		route("POST", "/traps/listen", false, func(r *http.Request) (any, error) {
			var body service.TrapSettings
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			return s.Listen(body)
		}),
		route("POST", "/traps/stop", false, func(*http.Request) (any, error) { return s.StopListening(), nil }),
	}
}

func (a *Area) mibRoutes() []api.Route {
	library := a.snmp.MIBs
	return []api.Route{
		route("GET", "/mibs", false, func(*http.Request) (any, error) {
			return map[string]any{"modules": library.Modules()}, nil
		}),
		route("POST", "/mibs", false, func(r *http.Request) (any, error) {
			var body struct {
				Name string `json:"name"`
				Text string `json:"text"`
			}
			if err := decode(r, &body); err != nil {
				return nil, err
			}
			modules, err := library.Import(body.Name, []byte(body.Text))
			return map[string]any{"imported": modules, "modules": library.Modules()}, err
		}),
		route("DELETE", "/mibs/{module}", false, func(r *http.Request) (any, error) {
			if err := library.Remove(r.PathValue("module")); err != nil {
				return nil, err
			}
			return map[string]any{"modules": library.Modules()}, nil
		}),
		route("POST", "/mibs/restore", false, func(*http.Request) (any, error) {
			if err := library.Restore(); err != nil {
				return nil, err
			}
			return map[string]any{"modules": library.Modules()}, nil
		}),
		route("GET", "/mibs/resolve", false, func(r *http.Request) (any, error) {
			name := r.URL.Query().Get("name")
			oid, found := library.Lib().Resolve(name)
			if !found {
				return nil, apperr.ValueError("No MIB knows " + name + "; type the numeric OID or import its MIB")
			}
			return map[string]any{"oid": oid, "name": library.Name(oid)}, nil
		}),
	}
}
