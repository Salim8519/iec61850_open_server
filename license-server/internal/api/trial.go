package api

import (
	"errors"
	"net/http"

	"substation-license-server/internal/store"
	"substation-license-server/internal/trialemail"
)

// trial grants one free trial per machine. Clear proof (same machine ID, or every
// usable hardware part of an ended trial) refuses; a suspicion (2 of 3 parts, a busy
// network) only asks for a confirmed email, one trial per address.
func (s *Server) trial(r *http.Request, ip string) (any, error) {
	request, err := s.readMachine(r, false, true)
	if err != nil {
		return nil, err
	}
	if s.dataFull() {
		return nil, errTrialsPaused
	}
	emailHash := ""
	if request.Email != "" {
		if emailHash, err = s.confirmEmail(request); err != nil {
			s.Log.Event("trial_refused", map[string]any{"ip": ip, "hwid": request.HWID[:12], "reason": "bad_code"})
			return nil, err
		}
	}
	trial, outcome, err := s.Store.StartTrialFor(store.TrialRequest{HWID: request.HWID, Parts: request.Parts, IP: ip,
		EmailHash: emailHash, TrialDays: s.Config.TrialDays, IPEmailAfter: s.Config.TrialIPEmailAfter,
		IPWindow: s.Config.TrialIPWindow()}, s.Now())
	var refused *store.TrialRefused
	if errors.As(err, &refused) {
		// Never a full machine ID, part hash, key or email in the log.
		s.Log.Event("trial_refused", map[string]any{"ip": ip, "hwid": request.HWID[:12],
			"reason": refused.Reason, "signals": refused.Matched, "parts": request.Parts.Clean().Count()})
	}
	if err != nil {
		return nil, err
	}
	bundle, err := s.trialCertificate(trial)
	s.Log.Event("trial", map[string]any{"ip": ip, "hwid": request.HWID[:12], "outcome": outcome,
		"parts": request.Parts.Clean().Count(), "email": emailHash != ""})
	return s.withTime(map[string]any{"certificate": bundle}), err
}

// confirmEmail checks the code sent by trialEmailCode and returns the address hash.
func (s *Server) confirmEmail(request machineRequest) (string, error) {
	normalized, err := trialemail.Normalize(request.Email)
	if err != nil {
		return "", apiError{http.StatusBadRequest, "bad_email", err.Error()}
	}
	hash := trialemail.Hash(normalized)
	want := trialemail.MAC(s.Config.PowSecret, request.HWID, hash, request.EmailCode)
	err = s.Store.ConsumeEmailCode(request.HWID, hash, func(mac string) bool { return trialemail.Same(mac, want) }, s.Now())
	return hash, err
}

// withTime adds the signed server time so trial apps can anchor their clock. A
// signing failure leaves it out; the app then simply keeps its local anchor.
func (s *Server) withTime(answer map[string]any) map[string]any {
	now := s.Now().Unix()
	answer["time"] = now
	stamp, err := s.Signer.SignTime(now)
	if err != nil {
		s.Log.Event("time_stamp_error", map[string]any{"error": err.Error()})
		return answer
	}
	answer["server_time"] = stamp
	return answer
}
