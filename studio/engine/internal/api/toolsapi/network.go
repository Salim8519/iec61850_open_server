package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
)

// NetworkRoutes are the Network fields only the Go engine serves: the SCADA master's (ADR-0123)
// and the station gateway's (ADR-0124). The frozen Python engine answers them 404.
var NetworkRoutes = map[string]bool{"GET /api/v1/tools/master/network": true, "PUT /api/v1/tools/master/network": true,
	"GET /api/v1/tools/gateway/network": true, "PUT /api/v1/tools/gateway/network": true,
	"POST /api/v1/tools/gateway/network/own": true}

// networkRoutes read and save the Network fields. Master: "local" is an address of this PC or ""
// (All networks). Gateway: "choice" is this_pc, network or an address of this PC; "allowed" is the
// only SCADA address that may connect, or "" for anyone. Gateway own: "address" is typed by the
// user and goes on the ScadaBench Lab card (ADR-0125).
func (a *Area) networkRoutes() []api.Route {
	m, g := a.deps.Master, a.deps.Gateway
	return []api.Route{
		route("GET", "/tools/master/network", false, func(*http.Request) (any, error) { return m.Network(), nil }),
		route("PUT", "/tools/master/network", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			local, isText := value(data, "local", nil).(string)
			if !isText {
				return nil, apperr.ValueError("Send local: an address of this PC, or empty for All networks")
			}
			if err := m.SaveNetwork(local); err != nil {
				return nil, err
			}
			return m.Network(), nil
		}),
		route("GET", "/tools/gateway/network", false, func(*http.Request) (any, error) { return g.Network(), nil }),
		route("PUT", "/tools/gateway/network", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			choice, isText := value(data, "choice", nil).(string)
			allowed, isAllowed := value(data, "allowed", "").(string)
			if !isText || !isAllowed {
				return nil, apperr.ValueError("Send choice (this_pc, network or an address of this PC) and allowed")
			}
			if err := g.SaveNetwork(choice, allowed); err != nil {
				return nil, err
			}
			return g.Network(), nil
		}),
		route("POST", "/tools/gateway/network/own", false, func(r *http.Request) (any, error) {
			data, err := body(r)
			if err != nil {
				return nil, err
			}
			address, isText := value(data, "address", nil).(string)
			if !isText {
				return nil, apperr.ValueError("Send address: the gateway's own address, for example 10.10.0.100")
			}
			if err := g.AddOwnAddress(address); err != nil {
				return nil, err
			}
			return g.Network(), nil
		}),
	}
}
