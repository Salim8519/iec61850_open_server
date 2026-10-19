package api

import (
	"errors"
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
}

func (s *Server) readMachine(r *http.Request, needKey, needPow bool) (machineRequest, error) {
	var request machineRequest
	if err := decode(r, &request); err != nil {
		return request, err
	}
	request.HWID = strings.TrimSpace(request.HWID)
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

func (s *Server) activate(r *http.Request, ip string) (any, error) {
	request, err := s.readMachine(r, true, true)
	if err != nil {
		return nil, err
	}
	license, err := s.Store.Activate(request.LicenseKey, request.HWID, s.Now())
	if errors.Is(err, store.ErrBoundElsewhere) && s.Config.MaxTransfersPerYear > 0 {
		return nil, apiError{http.StatusConflict, "bound_elsewhere", "This key is active on another computer. " +
			"You can move it to this computer (the other computer stops working when it next checks online)."}
	}
	if err != nil {
		return nil, err
	}
	bundle, err := s.CertificateFor(license, request.HWID)
	s.Log.Event("activate", map[string]any{"ip": ip, "key": logging.MaskKey(license.Key), "hwid": request.HWID[:12]})
	return s.withTime(map[string]any{"certificate": bundle}), err
}

func (s *Server) transfer(r *http.Request, ip string) (any, error) {
	request, err := s.readMachine(r, true, true)
	if err != nil {
		return nil, err
	}
	license, err := s.Store.Transfer(request.LicenseKey, request.HWID, s.Now(), s.Config.MaxTransfersPerYear)
	if err != nil {
		return nil, err
	}
	bundle, err := s.CertificateFor(license, request.HWID)
	s.Log.Event("transfer", map[string]any{"ip": ip, "key": logging.MaskKey(license.Key), "hwid": request.HWID[:12]})
	return s.withTime(map[string]any{"certificate": bundle}), err
}

// verify lets installed apps learn about refunds, revocations and moves; every
// answer carries the signed server time that trial apps confirm once a day.
func (s *Server) verify(r *http.Request, ip string) (any, error) {
	request, err := s.readMachine(r, false, false)
	if err != nil {
		return nil, err
	}
	if request.LicenseKey == "" || strings.HasPrefix(request.LicenseKey, "TRIAL-") {
		return s.withTime(map[string]any{"status": "valid"}), nil
	}
	license, err := s.Store.GetLicense(request.LicenseKey)
	if errors.Is(err, store.ErrUnknownKey) {
		return s.withTime(map[string]any{"status": "unknown"}), nil
	}
	if err != nil {
		return nil, err
	}
	switch {
	case license.Revoked:
		return s.withTime(map[string]any{"status": "revoked"}), nil
	case license.HWID != request.HWID:
		return s.withTime(map[string]any{"status": "moved"}), nil
	}
	bundle, err := s.CertificateFor(license, request.HWID)
	return s.withTime(map[string]any{"status": "valid", "certificate": bundle}), err
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
