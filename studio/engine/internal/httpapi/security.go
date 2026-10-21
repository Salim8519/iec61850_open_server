package httpapi

import (
	"net/http"
	"net/netip"
	"strings"

	"studio-engine/internal/pyjson"
)

// protect is api/security.py protect_request; it answers and returns true when the
// request must stop.
func (s *Server) protect(w http.ResponseWriter, r *http.Request) bool {
	host := requestHost(r)
	name, _, _ := strings.Cut(host, ":")
	if (name != "127.0.0.1" && name != "localhost") || !loopbackCaller(r.RemoteAddr) {
		writeMessage(w, http.StatusForbidden, "Administration is available on loopback only")
		return true
	}
	origin, _ := Header(r, "Origin")
	devOrigin := s.config.AdminOrigin
	frontend, _ := Header(r, "X-Studio-Frontend")
	if frontend == "" {
		frontend = origin
	}
	wrong := devOrigin != "" && frontend != "" && strings.TrimRight(frontend, "/") != devOrigin
	if devOrigin == "" && origin != "" {
		netloc, err := urlNetloc(origin)
		if err != nil {
			WriteError(w, err)
			return true
		}
		wrong = netloc != host
	}
	if wrong {
		message := "Cross-origin administration is not permitted"
		var active any
		if devOrigin != "" {
			message = "This is not the active ScadaBench browser tab. Reopen ScadaBench from start-studio.cmd."
			active = devOrigin
		}
		WriteJSON(w, http.StatusForbidden, pyjson.NewObject().Set("error", message).Set("active_frontend", active))
		return true
	}
	if !strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api/v1/session" {
		return false
	}
	if !s.sessionAuthenticated(w, r) {
		authorization, _ := Header(r, "Authorization")
		equal, err := compareDigest(strings.TrimPrefix(authorization, "Bearer "), s.config.Token)
		if err != nil {
			WriteError(w, err)
			return true
		}
		if !equal {
			launcher := "ScadaBench.exe"
			if devOrigin != "" {
				launcher = "start-studio.cmd"
			}
			writeMessage(w, http.StatusUnauthorized, "This browser tab has no ScadaBench session. Open the full "+
				"address printed by "+launcher+" (it ends with #session=...).")
			return true
		}
	}
	return s.requireLicense(w, r)
}

// requireLicense is _require_license: 402 until licensed, only when enforced and outside
// /api/v1/license*.
func (s *Server) requireLicense(w http.ResponseWriter, r *http.Request) bool {
	gate := s.config.License
	if gate == nil || !gate.Enforced() || strings.HasPrefix(r.URL.Path, "/api/v1/license") || gate.Licensed() {
		return false
	}
	WriteJSON(w, http.StatusPaymentRequired, pyjson.NewObject().
		Set("error", "A license is required. Start the free trial or activate your key.").
		Set("code", "license_required"))
	return true
}

// requestHost is Werkzeug's request.host: the Host header, without ":80" for http.
func requestHost(r *http.Request) string {
	return strings.TrimSuffix(Latin1(r.Host), ":80")
}

// loopbackCaller is _loopback_caller: the peer address must be a loopback address.
func loopbackCaller(remote string) bool {
	if remote == "" {
		return true
	}
	if addrPort, err := netip.ParseAddrPort(remote); err == nil {
		return addrPort.Addr().Unmap().IsLoopback()
	}
	address, err := netip.ParseAddr(remote)
	return err == nil && address.IsLoopback()
}

// noWait lists the prefixes the startup gate never holds (api/startup_gate.py NO_WAIT).
var noWait = []string{"/api/v1/session", "/api/v1/license", "/api/v1/preferences"}

// waitForStartup is startup_gate.wait_for_project.
func (s *Server) waitForStartup(w http.ResponseWriter, r *http.Request) bool {
	gate := s.config.Startup
	if gate == nil || !strings.HasPrefix(r.URL.Path, "/api/") {
		return false
	}
	for _, prefix := range noWait {
		if strings.HasPrefix(r.URL.Path, prefix) {
			return false
		}
	}
	if !gate.Wait(s.config.StartupWait) {
		WriteJSON(w, http.StatusServiceUnavailable, pyjson.NewObject().
			Set("error", "ScadaBench is still opening your last project. Try again in a moment.").
			Set("code", "project_opening"))
		return true
	}
	if err := gate.Err(); err != nil {
		WriteJSON(w, http.StatusServiceUnavailable, pyjson.NewObject().
			Set("error", "ScadaBench stopped to protect your last autosave: "+err.Error()+
				". Close ScadaBench, inspect the data folder, then start it again.").
			Set("code", "startup_recovery_failed"))
		return true
	}
	return false
}
