// Package platformapi serves the platform routes of api/license.py, api/updates.py and the
// lab-network part of api/network.py (license status and activation, the update check, the
// network mode, adapter plan/apply/clean-up and the gateway firewall) with Flask's exact
// paths, status codes, JSON bytes and error texts.
//
// Every service is reached through a small interface in Deps, so the integrator wires the
// real licensing.Service, updates.Service, netcfg.NetworkService, netcfg.GatewayFirewall,
// the project service (ProjectNetwork) and the coordinator, and tests use fakes. Nothing
// here changes an adapter, a firewall rule or asks for elevation by itself: the network
// service does that only after the user confirmed an exact preview.
package platformapi

import (
	"net/http"
	"sync"

	"studio-engine/internal/api"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

// License is LicenseService as api/license.py uses it (*licensing.Service).
type License interface {
	Status() (*pyjson.Object, error)
	StartTrial() (*pyjson.Object, error)
	Activate(licenseKey any, move bool) (*pyjson.Object, error)
	ActivateOffline(certificate any) (*pyjson.Object, error)
	SetServerURL(url any) (*pyjson.Object, error)
	// CertFile is the license.cert path; used only to word an unreadable file's error
	// exactly as Python's UnicodeDecodeError did (see errors.go).
	CertFile() string
}

// Updates is UpdateService as api/updates.py uses it (*updates.Service).
type Updates interface {
	Status() (*pyjson.Object, error)
	Check(manual bool) (*pyjson.Object, error)
	Configure(changes any) (*pyjson.Object, error)
}

// Network is NetworkService as api/network.py uses it (*netcfg.NetworkService).
type Network interface {
	Adapters(refresh bool) ([]netcfg.Adapter, error)
	Preview(guid any, strategy string, manual any) (*pyjson.Object, error)
	Apply(fingerprint, confirm any) (*pyjson.Object, error)
	CleanupPreview() (*pyjson.Object, error)
	Cleanup(fingerprint, confirm any) (*pyjson.Object, error)
	// PreviewProgress is the live state of the running address check (Go only, change 0150).
	PreviewProgress() *pyjson.Object
}

// Firewall is GatewayFirewall (*netcfg.GatewayFirewall).
type Firewall interface {
	Preview() (*pyjson.Object, error)
	Apply(fingerprint, confirm any) (*pyjson.Object, error)
}

// Projects is what the network-mode routes need from the project service (ProjectNetwork
// adapts *service.Service).
type Projects interface {
	// Network is project["network"].
	Network() any
	// UpdateNetwork is ied_settings.update_network: checks, stores, saves and announces.
	UpdateNetwork(value any) (*pyjson.Object, error)
}

// Runtime is the coordinator: its state and cycle_lock.
type Runtime interface {
	State() string
	CycleLock() sync.Locker
}

// Deps are the services behind the routes.
type Deps struct {
	License    License
	Updates    Updates
	Network    Network
	Firewall   Firewall
	Projects   Projects
	Runtime    Runtime
	AppVersion string // studio_app.version.__version__, e.g. "0.1.14+dev"
	// Npcap is npcap.npcap_status(); nil means netcfg.NpcapStatus("") (this PC).
	Npcap func() *pyjson.Object
}

// Area is the platform route set.
type Area struct{ deps Deps }

// New returns the area for deps.
func New(deps Deps) *Area {
	if deps.Npcap == nil {
		deps.Npcap = func() *pyjson.Object { return netcfg.NpcapStatus("") }
	}
	return &Area{deps: deps}
}

// handler answers a JSON value with 200, or the error the way Flask's handlers did.
type handler func(r *http.Request) (any, error)

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	value, err := run(h, r)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

// GoOnlyRoutes are the routes this area adds beyond Flask's.
var GoOnlyRoutes = map[string]bool{"GET /api/v1/network/plan/progress": true}

// Routes implements api.Area: 5 license, 3 update and 10 network routes (one Go only).
func (a *Area) Routes() []api.Route {
	route := func(method, pattern string, h handler) api.Route {
		return api.Route{Method: method, Pattern: "/api/v1" + pattern, Handler: h}
	}
	return []api.Route{
		route("GET", "/license/status", a.licenseStatus),
		route("POST", "/license/trial", a.licenseTrial),
		route("POST", "/license/activate", a.licenseActivate),
		route("POST", "/license/activate-offline", a.licenseActivateOffline),
		route("PUT", "/license/config", a.licenseConfig),
		route("GET", "/app/update", a.updateStatus),
		route("POST", "/app/update/check", a.updateCheck),
		route("PUT", "/app/update", a.updateConfigure),
		route("GET", "/network", a.networkStatus),
		route("GET", "/network/adapters", a.networkAdapters),
		route("PUT", "/network", a.networkMode),
		route("POST", "/network/plan/preview", a.planPreview),
		route("POST", "/network/plan/apply", a.planApply),
		route("GET", "/network/plan/progress", a.planProgress),
		route("GET", "/network/cleanup/preview", a.cleanupPreview),
		route("POST", "/network/cleanup/apply", a.cleanupApply),
		route("GET", "/network/gateway-firewall", a.firewallPreview),
		route("POST", "/network/gateway-firewall", a.firewallApply),
	}
}
