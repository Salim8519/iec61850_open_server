package api

import (
	"net/http"
	"strings"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/logging"
	"substation-license-server/internal/security"
	"substation-license-server/internal/store"
)

type machineRequest struct {
	security.Solution
	HWID       string           `json:"hwid"`
	LicenseKey string           `json:"license_key"`
	Parts      store.TrialParts `json:"parts"`
	Email      string           `json:"email"`
	EmailCode  string           `json:"email_code"`
	// Nonce is optional and only read by verify: the signed answer echoes it.
	Nonce string `json:"nonce"`
	// SentKey is license_key exactly as the app sent it (LicenseKey is normalised).
	SentKey string `json:"-"`
}

func (s *Server) readMachine(r *http.Request, needKey, needPow bool) (machineRequest, error) {
	var request machineRequest
	if err := decode(r, &request); err != nil {
		return request, err
	}
	request.HWID = strings.TrimSpace(request.HWID)
	request.SentKey = request.LicenseKey
	request.LicenseKey = certs.NormalizeKey(request.LicenseKey)
	if !certs.ValidHWID(request.HWID) {
		return request, badRequest("Machine ID must be the 64-character code shown in ScadaBench")
	}
	if needKey && !strings.HasPrefix(request.LicenseKey, "SUBS-") {
		return request, badRequest("License key must look like SUBS-XXXX-XXXX-XXXX-XXXX")
	}
	if needPow {
		return request, s.checkPow(request.Solution)
	}
	return request, nil
}

// CertificateFor signs the certificate a license record entitles hwid to.
func (s *Server) CertificateFor(license store.License, hwid string) (certs.Bundle, error) {
	now := s.Now().Unix()
	expires := int64(0)
	if license.Type == "trial" {
		days := license.Days
		if days <= 0 {
			days = s.Config.TrialDays
		}
		expires = license.ActivatedAt + int64(days)*86400
	}
	return s.Signer.Sign(certs.Payload{Email: license.Email, ExpiresAt: expires, HWID: hwid,
		IssuedAt: now, LicenseID: license.Key, MaxMachines: 1, Type: license.Type})
}

func (s *Server) trialCertificate(trial store.Trial) (certs.Bundle, error) {
	return s.Signer.Sign(certs.Payload{ExpiresAt: trial.ExpiresAt, HWID: trial.HWID,
		IssuedAt: trial.IssuedAt, LicenseID: "TRIAL-" + trial.HWID[:12], MaxMachines: 1, Type: "trial"})
}

// activate binds an unbound key to this machine, or confirms the same machine again. A key
// bound to another machine stays there: every license works on one computer only.
func (s *Server) activate(r *http.Request, ip string) (any, error) {
	request, err := s.readMachine(r, true, true)
	if err != nil {
		return nil, err
	}
	license, err := s.Store.Activate(request.LicenseKey, request.HWID, s.Now())
	if err != nil {
		return nil, err
	}
	bundle, err := s.CertificateFor(license, request.HWID)
	s.Log.Event("activate", map[string]any{"ip": ip, "key": logging.MaskKey(license.Key), "hwid": request.HWID[:12]})
	return s.withTime(map[string]any{"certificate": bundle}), err
}

// errMovesDisabled is what old 0.1.x apps (which still offer "move the key") read.
var errMovesDisabled = apiError{http.StatusGone, "moves_disabled",
	"Moving a license to another computer is not possible. Each license works on one computer only."}

// transfer is retired (ADR-0094): it never reads the request and never changes a key.
func (s *Server) transfer(_ *http.Request, ip string) (any, error) {
	s.Log.Event("transfer_refused", map[string]any{"ip": ip})
	return nil, errMovesDisabled
}

// IssueKey creates an admin key; with hwid it is pre-bound for offline delivery.
func (s *Server) IssueKey(email, kind string, days int, hwid string) (store.License, *certs.Bundle, error) {
	if kind != "trial" {
		kind = "lifetime"
	}
	now := s.Now()
	license := store.License{Key: certs.NewLicenseKey(), Email: strings.TrimSpace(email), Type: kind,
		Days: days, CreatedAt: now.Unix()}
	if hwid != "" {
		if !certs.ValidHWID(hwid) {
			return license, nil, badRequest("hwid must be 64 lowercase hex characters")
		}
		license.HWID, license.ActivatedAt = hwid, now.Unix()
	}
	if err := s.Store.CreateLicense(license); err != nil {
		return license, nil, err
	}
	s.Log.Event("issue", map[string]any{"key": logging.MaskKey(license.Key), "type": kind})
	if hwid == "" {
		return license, nil, nil
	}
	bundle, err := s.CertificateFor(license, hwid)
	return license, &bundle, err
}
