package licensing

import (
	"errors"
	"os"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// evaluate is LicenseService._evaluate: integrity, revoked reason, certificate, policy.
func (s *Service) evaluate() (Status, error) {
	if !s.options.Enforce {
		return DevelopmentStatus(), nil
	}
	folder := s.options.NativeFolder
	if folder == "" {
		folder = "."
	}
	if problem := IntegrityProblem(folder, s.options.ExpectedFiles, s.options.Debugger); problem != "" {
		return Unlicensed(problem, nil, nil, nil), nil
	}
	s.revoked.Restore() // a deleted copy of the cancelled IDs comes back from the others
	state, err := readObject(s.stateFile)
	if err != nil {
		return Status{}, err
	}
	revoked, _ := state.Get("revoked_reason")
	text, err := pytext.ReadText(os.ReadFile(s.certFile))
	if errors.Is(err, pytext.ErrUTF8) {
		return Status{}, uncaught("license.cert is not UTF-8")
	}
	if err != nil {
		if !truthy(revoked) {
			revoked = NoLicense
		}
		return Unlicensed(revoked, nil, nil, nil), nil
	}
	payload, err := s.verifyText(text)
	var certificate *CertificateError
	if errors.As(err, &certificate) {
		return Unlicensed(certificate.Message, nil, nil, nil), nil
	}
	if err != nil {
		return Status{}, err
	}
	if licenseID, _ := payload.Get("license_id"); s.revoked.Contains(licenseID) {
		plan, _ := payload.Get("type") // cancelled stays cancelled, whatever the file says
		return Unlicensed(RevokedText, plan, licenseID, nil), nil
	}
	now := s.now()
	bound, err := s.HWID()
	if err != nil {
		return Status{}, err
	}
	var status Status
	if plan, _ := payload.Get("type"); plan != any("trial") { // lifetime: the original rule plus its binding
		status, err = s.evaluateLifetime(payload, bound, now)
	} else {
		status, err = s.evaluateTrial(payload, bound, now)
	}
	if err != nil {
		return Status{}, err
	}
	if status.Licensed {
		s.anchor.Advance(now)
	}
	return status, nil
}

// evaluateLifetime is the lifetime rule of license_policy (300 s clock tolerance) plus what
// ties the license to this computer (ADR-0098): the security chip's proof, or the online lease.
func (s *Service) evaluateLifetime(payload *pyjson.Object, bound string, now int64) (Status, error) {
	seen := s.anchor.MaxSeen()
	if leaseDays(payload) > 0 && certAK(payload) == "" {
		seen = max(seen, s.lease.ReceiptTime(payload, bound)) // signed server time guards the clock
	}
	status, err := EvaluateCertificate(payload, bound, now, seen, false, ClockToleranceSeconds)
	if err != nil {
		return Status{}, uncaught(err.Error())
	}
	if !status.Licensed {
		return status, nil
	}
	return s.applyBinding(status, payload, bound, now), nil
}

// evaluateTrial is LicenseService._evaluate_trial: a missing anchor only asks for one
// quiet online check; the rollback reference includes the signed server time.
func (s *Service) evaluateTrial(payload *pyjson.Object, bound string, now int64) (Status, error) {
	confirmed, err := s.trialClock.Confirmed()
	if err != nil {
		return Status{}, err
	}
	if AnchorMissing(len(s.anchor.Value.ReadInts()), confirmed) {
		s.trialClock.MarkSuspicious(now) // never blocks now
	}
	status, err := EvaluateCertificate(payload, bound, now, LatestSeen(s.anchor.MaxSeen(), confirmed), false,
		TrialClockToleranceSeconds)
	if err != nil {
		return Status{}, uncaught(err.Error())
	}
	suspicious := s.trialClock.SuspiciousSince()
	host, err := s.ServerHost()
	if err != nil {
		return Status{}, err
	}
	return ApplyTrial(status, confirmed, suspicious, now, host), nil
}

func (s *Service) verifyText(input any) (*pyjson.Object, error) {
	bundle, err := ParseBundle(input)
	if err != nil {
		return nil, err
	}
	return VerifyBundle(bundle, s.options.Keys)
}

// Current is LicenseService.current: the cached status, re-evaluated when refresh is
// set or the cache is older than StatusCacheSeconds (or the clock went backwards).
// Evaluation runs outside the lock (Go rule: no lock across I/O); concurrent refreshes
// at worst evaluate twice.
func (s *Service) Current(refresh bool) (Status, error) {
	s.mu.Lock()
	age := s.options.Clock() - s.cachedAt
	if !refresh && s.cached != nil && age >= 0 && age < StatusCacheSeconds {
		cached := *s.cached
		s.mu.Unlock()
		return cached, nil
	}
	s.mu.Unlock()
	status, err := s.evaluate()
	if err != nil {
		return Status{}, err
	}
	s.mu.Lock()
	s.cached, s.cachedAt = &status, s.options.Clock()
	s.mu.Unlock()
	return status, nil
}

// invalidate drops the cached status (Python: self._cached = None).
func (s *Service) invalidate() {
	s.mu.Lock()
	s.cached = nil
	s.mu.Unlock()
}

// Enforced reports whether this build checks licenses (Gate).
func (s *Service) Enforced() bool { return s.options.Enforce }

// Licensed is LicenseService.is_licensed (Gate).
func (s *Service) Licensed() (bool, error) {
	status, err := s.Current(false)
	return status.Licensed, err
}

// Status is LicenseService.status(): the policy fields plus enforced, server_url,
// masked_key, hwid and transfer_available (always false: licenses never move, ADR-0097),
// in Python's key order.
func (s *Service) Status() (*pyjson.Object, error) {
	current, err := s.Current(true)
	if err != nil {
		return nil, err
	}
	result := current.Object()
	serverURL, hwid := "", any(nil)
	if s.options.Enforce {
		if serverURL, err = s.ServerURL(); err != nil {
			return nil, err
		}
		if hwid, err = s.HWID(); err != nil {
			return nil, err
		}
	}
	return result.Set("enforced", s.options.Enforce).Set("server_url", serverURL).
		Set("masked_key", MaskKey(current.LicenseID)).Set("hwid", hwid).
		Set("transfer_available", false), nil
}

// StatusBody is the GET /api/v1/license/status body: Status() plus app_version, as
// Flask's jsonify writes it.
func (s *Service) StatusBody(appVersion string) ([]byte, error) {
	status, err := s.Status()
	if err != nil {
		return nil, err
	}
	return pyjson.Flask(status.Set("app_version", appVersion))
}
