package api

import (
	"crypto/subtle"
	"net"
	"net/http"
	"strings"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/store"
	"substation-license-server/internal/trialemail"
)

// AdminRequest is shared by the HTTP admin API and the CLI fallback.
type AdminRequest struct {
	Email  string `json:"email"`
	Type   string `json:"type"`
	Days   int    `json:"days"`
	HWID   string `json:"hwid"`
	Key    string `json:"key"`
	Reason string `json:"reason"`
	// IP, Parts and Email serve the trial overrides (reset-trial-ip, reset-trial-parts,
	// allow-email). The email is only hashed, never stored or logged in clear.
	IP         string           `json:"ip"`
	Parts      store.TrialParts `json:"parts"`
	TrialEmail string           `json:"trial_email"`
}

// adminAllowed requires the bearer token AND a direct local connection: anything
// arriving through the Cloudflare tunnel carries CF-Connecting-IP and is refused.
func (s *Server) adminAllowed(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	peer := net.ParseIP(host)
	if err != nil || peer == nil || !peer.IsLoopback() || r.Header.Get("CF-Connecting-IP") != "" {
		return false
	}
	supplied := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return s.Config.AdminToken != "" &&
		subtle.ConstantTimeCompare([]byte(supplied), []byte(s.Config.AdminToken)) == 1
}

func (s *Server) admin(w http.ResponseWriter, r *http.Request) {
	if !s.adminAllowed(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin access denied", "code": "forbidden"})
		return
	}
	if r.PathValue("action") == "backup" {
		w.Header().Set("Content-Type", "application/octet-stream")
		if err := s.Store.Backup(w); err != nil {
			s.Log.Event("backup_error", map[string]any{"error": err.Error()})
		}
		return
	}
	var request AdminRequest
	if err := decode(r, &request); err != nil {
		s.writeError(w, r, "admin", err)
		return
	}
	result, err := s.RunAdmin(r.PathValue("action"), request)
	if err != nil {
		s.writeError(w, r, "admin", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// RunAdmin executes one admin action against the store.
func (s *Server) RunAdmin(action string, request AdminRequest) (any, error) {
	key := certs.NormalizeKey(request.Key)
	switch action {
	case "issue":
		license, bundle, err := s.IssueKey(request.Email, request.Type, request.Days, strings.TrimSpace(request.HWID))
		if err != nil {
			return nil, err
		}
		return map[string]any{"license": license, "certificate": bundle}, nil
	case "revoke":
		reason := request.Reason
		if reason == "" {
			reason = "admin"
		}
		return s.Store.Revoke(key, reason)
	case "reset-hwid":
		return s.Store.ResetHWID(key)
	case "reset-trial":
		hwid := strings.TrimSpace(request.HWID)
		if !certs.ValidHWID(hwid) {
			return nil, badRequest("--hwid must be the 64-character Machine ID")
		}
		found, err := s.Store.ResetTrial(hwid)
		if err != nil {
			return nil, err
		}
		// Parts given too (license_admin.py sends them): also forget this hardware.
		known, err := s.Store.ResetTrialParts(request.Parts)
		s.Log.Event("admin_reset_trial", map[string]any{"hwid": hwid[:12], "parts": known})
		return map[string]any{"hwid": hwid, "had_trial": found, "parts_cleared": known}, err
	case "reset-trial-parts":
		if request.Parts.Clean().Count() == 0 {
			return nil, badRequest("give at least one part hash (--guid-hash, --volume-hash, --firmware-hash)")
		}
		known, err := s.Store.ResetTrialParts(request.Parts)
		s.Log.Event("admin_reset_trial_parts", map[string]any{"parts": known})
		return map[string]any{"parts_cleared": known}, err
	case "reset-trial-ip":
		ip := strings.TrimSpace(request.IP)
		if net.ParseIP(ip) == nil {
			return nil, badRequest("--ip must be an IP address, for example 203.0.113.7")
		}
		found, err := s.Store.ResetTrialIP(ip)
		s.Log.Event("admin_reset_trial_ip", map[string]any{"ip": ip})
		return map[string]any{"ip": ip, "had_count": found}, err
	case "allow-email":
		normalized, err := trialemail.Normalize(request.TrialEmail)
		if err != nil {
			return nil, badRequest(err.Error())
		}
		found, err := s.Store.AllowEmail(trialemail.Hash(normalized))
		s.Log.Event("admin_allow_email", map[string]any{"had_trial": found})
		return map[string]any{"email_had_trial": found}, err
	case "delete":
		if err := s.Store.DeleteLicense(key); err != nil {
			return nil, err
		}
		return map[string]any{"deleted": key}, nil
	case "list":
		licenses, err := s.Store.ListLicenses()
		if err != nil {
			return nil, err
		}
		busy, err := s.Store.BusyNetworks(s.Config.TrialIPEmailAfter, s.Config.TrialIPWindow(), s.Now())
		return map[string]any{"licenses": licenses, "trial_machines": s.Store.TrialCount(), "busy_networks": busy}, err
	case "certificate":
		license, err := s.Store.GetLicense(key)
		if err != nil {
			return nil, err
		}
		if license.HWID == "" {
			return nil, badRequest("key is not activated on any machine yet")
		}
		if license.Revoked {
			return nil, store.ErrRevoked
		}
		return s.CertificateFor(license, license.HWID)
	}
	return nil, apiError{http.StatusNotFound, "unknown_action", "unknown admin action " + action}
}
