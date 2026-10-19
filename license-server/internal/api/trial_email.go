package api

import (
	"context"
	"net/http"
	"time"

	"substation-license-server/internal/store"
	"substation-license-server/internal/trialemail"
)

const emailCodeLifetime = 15 * time.Minute

// trialEmailCode sends a 6-digit code to the address a suspicious trial must confirm.
// If the mail cannot be sent (Mailgun down or not set up) the step is waived and
// logged: errors never count against the user, but the address is still one-per-trial.
func (s *Server) trialEmailCode(r *http.Request, ip string) (any, error) {
	request, err := s.readMachine(r, false, true)
	if err != nil {
		return nil, err
	}
	normalized, err := trialemail.Normalize(request.Email)
	if err != nil {
		return nil, apiError{http.StatusBadRequest, "bad_email", err.Error()}
	}
	if trialemail.Disposable(trialemail.Domain(normalized)) {
		s.Log.Event("trial_refused", map[string]any{"ip": ip, "hwid": request.HWID[:12], "reason": "disposable_email"})
		return nil, apiError{http.StatusBadRequest, "email_disposable",
			"Throwaway email services cannot be used for the free trial. Use your work or personal email."}
	}
	hash := trialemail.Hash(normalized)
	if used, err := s.Store.EmailUsed(hash); err != nil {
		return nil, err
	} else if used {
		s.Log.Event("trial_refused", map[string]any{"ip": ip, "hwid": request.HWID[:12], "reason": "email_used"})
		return nil, store.ErrEmailUsed
	}
	code := trialemail.NewCode()
	pending := store.EmailCode{EmailHash: hash, MAC: trialemail.MAC(s.Config.PowSecret, request.HWID, hash, code),
		Expires: s.Now().Add(emailCodeLifetime).Unix()}
	status := "sent"
	if err := s.sendTrialCode(request.Email, code); err != nil {
		pending.Waived, status = true, "no_code_needed"
		s.Log.Event("trial_email_unsent", map[string]any{"ip": ip, "hwid": request.HWID[:12], "error": err.Error()})
	}
	if err := s.Store.SaveEmailCode(request.HWID, pending); err != nil {
		return nil, err
	}
	s.Log.Event("trial_email_code", map[string]any{"ip": ip, "hwid": request.HWID[:12], "status": status})
	return map[string]any{"status": status}, nil
}

func (s *Server) sendTrialCode(to, code string) error {
	if s.Mailer == nil {
		return errMailOff
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	text := "Your ScadaBench free-trial code is " + code + ".\n\n" +
		"Type it in ScadaBench to start the trial. The code works for 15 minutes.\n" +
		"If you did not ask for it, you can ignore this email.\n"
	return s.Mailer.Send(ctx, to, "Your ScadaBench trial code: "+code, text)
}

var errMailOff = apiError{http.StatusServiceUnavailable, "mail_off", "email delivery is not set up"}
