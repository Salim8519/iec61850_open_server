package toolsapi

import (
	"net/http"

	"studio-engine/internal/api"
	"studio-engine/internal/apperr"
)

// masterNetworkRoutes read and save the master's "Network" (ADR-0123, Go engine only): which
// address of this PC the master connects from; "" is All networks (Windows chooses).
func (a *Area) masterNetworkRoutes() []api.Route {
	m := a.deps.Master
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
	}
}
