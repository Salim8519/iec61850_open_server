package licensing

import (
	"errors"
	"time"

	"studio-engine/internal/pyjson"
)

// Background license checks (licensing/online_check.py): the weekly revocation check for
// lifetime keys (unchanged) and a quiet daily signed-time confirmation for trials.
// Failures never count against the user: an offline PC simply keeps its license.

// BackgroundCheckSeconds is how often the background loop asks about a trial.
const BackgroundCheckSeconds = 30 * 60

// BadTimeText is the error of a server answer without a genuine signed time.
const BadTimeText = "The license server answered without a valid signed time. Try again later. Help: " + SupportURL

// VerifyOnlineIfDue learns about refunds or key moves (lifetime, weekly) or confirms a
// trial's date (daily, and at once after the anchor went missing); it never fails
// offline. trialOnly skips the lifetime check. It returns "confirmed", the server's
// lifetime verdict, or nil when nothing was asked or the server could not be reached.
func (s *Service) VerifyOnlineIfDue(trialOnly bool) (any, error) {
	if !s.options.Enforce {
		return nil, nil
	}
	status, err := s.Current(true)
	if err != nil {
		return nil, err
	}
	if status.Plan == "trial" && truthy(status.LicenseID) && (status.Licensed || status.OnlineCheckRequired) {
		confirmed, err := s.trialClock.Confirmed()
		if err != nil {
			return nil, err
		}
		if !CheckDue(confirmed, s.trialClock.SuspiciousSince(), s.now()) {
			return nil, nil
		}
		result, err := s.confirmTrialTime(status)
		var server *ServerError
		if errors.As(err, &server) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	if trialOnly {
		return nil, nil
	}
	return s.verifyLifetime(status)
}

// CheckOnlineNow is the "Check online now" button: confirm the trial date immediately
// (server errors are returned for the user to read), then answer the status.
func (s *Service) CheckOnlineNow() (*pyjson.Object, error) {
	if err := s.requireEnforced(); err != nil {
		return nil, err
	}
	status, err := s.Current(true)
	if err != nil {
		return nil, err
	}
	if status.Plan == "trial" && truthy(status.LicenseID) {
		if _, err := s.confirmTrialTime(status); err != nil {
			return nil, err
		}
	}
	return s.Status()
}

// confirmTrialTime asks the server about the trial and remembers its signed time.
func (s *Service) confirmTrialTime(status Status) (any, error) {
	bound, err := s.HWID()
	if err != nil {
		return nil, err
	}
	verdict, answer, err := s.signedVerdict(status.LicenseID, bound)
	if err != nil {
		return nil, err
	}
	stamp, _ := answer.Get("server_time")
	remembered, err := s.trialClock.Remember(stamp)
	if err != nil {
		return nil, err
	}
	if !remembered {
		return nil, &ServerError{Message: BadTimeText, Code: "bad_time"}
	}
	if verdict == "revoked" || verdict == "moved" {
		reason := s.cancel(status.LicenseID, verdict)
		if err := s.updateState(pyjson.NewObject().Set("revoked_reason", reason)); err != nil {
			return nil, err
		}
	}
	s.invalidate()
	return "confirmed", nil
}

// StartBackgroundVerify runs the checks in the background when enforcing: once at start
// (lifetime keys, as before), then every BackgroundCheckSeconds for trials. Failures are
// logged, never shown.
func (s *Service) StartBackgroundVerify() {
	if !s.options.Enforce {
		return
	}
	go func() {
		s.safeVerify(false)
		ticker := time.NewTicker(BackgroundCheckSeconds * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			s.safeVerify(true)
		}
	}()
}

func (s *Service) safeVerify(trialOnly bool) {
	if _, err := s.VerifyOnlineIfDue(trialOnly); err != nil {
		s.options.Logf("Background license check failed: %v", err)
	}
}
