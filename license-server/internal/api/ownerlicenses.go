package api

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"time"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/logging"
	"substation-license-server/internal/mail"
	"substation-license-server/internal/store"
)

// ownerDailyKeys caps hand-made keys per 24 hours, so even a stolen session
// cannot hand out keys in bulk before the owner notices the sign-in email.
const ownerDailyKeys = 25

type ownerKeyRequest struct {
	Key    string `json:"key"`
	Email  string `json:"email"`
	Note   string `json:"note"`
	Reason string `json:"reason"`
	Send   bool   `json:"send_email"`
}

func readKeyRequest(r *http.Request) (ownerKeyRequest, error) {
	var request ownerKeyRequest
	if err := decode(r, &request); err != nil {
		return request, err
	}
	request.Key = certs.NormalizeKey(request.Key)
	request.Email = strings.TrimSpace(request.Email)
	request.Note = strings.TrimSpace(request.Note)
	if len([]rune(request.Note)) > 120 {
		return request, badRequest("Keep the note under 120 characters.")
	}
	return request, nil
}

func (s *Server) ownerLicenses(r *http.Request, _ *ownerCall) (any, error) {
	licenses, err := s.Store.ListLicenses()
	if err != nil {
		return nil, err
	}
	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	found := []store.License{}
	for _, license := range licenses {
		text := strings.ToLower(license.Key + " " + license.Email + " " + license.Note + " " + license.OrderID)
		if query == "" || strings.Contains(text, query) {
			found = append(found, license)
		}
	}
	sort.Slice(found, func(i, j int) bool { return found[i].CreatedAt > found[j].CreatedAt })
	total := len(found)
	if len(found) > 300 {
		found = found[:300]
	}
	return map[string]any{"licenses": found, "total": total, "mail_ready": s.Mailer != nil}, nil
}

func (s *Server) ownerIssue(r *http.Request, call *ownerCall) (any, error) {
	request, err := readKeyRequest(r)
	if err != nil {
		return nil, err
	}
	if request.Email != "" && !store.ValidEmail(request.Email) {
		return nil, badRequest("That email address does not look right.")
	}
	if request.Send && request.Email == "" {
		return nil, badRequest("Add their email address to send the key.")
	}
	if s.handMadeToday() >= ownerDailyKeys {
		return nil, apiError{http.StatusTooManyRequests, "daily_cap",
			"You made 25 keys in the last 24 hours, the safety limit. Try again tomorrow."}
	}
	license, _, err := s.IssueKey(request.Email, "lifetime", 0, "")
	if err != nil {
		return nil, err
	}
	if request.Note != "" {
		license, err = s.Store.UpdateLicense(license.Key, func(l *store.License) error { l.Note = request.Note; return nil })
		if err != nil {
			return nil, err
		}
	}
	s.ownerAudit("issue", call.ip, logging.MaskKey(license.Key)+" "+request.Note, true)
	return map[string]any{"license": license, "emailed": request.Send && s.sendKey(license) == nil}, nil
}

// handMadeToday counts keys created without a PayPal order in the last 24 hours.
func (s *Server) handMadeToday() int {
	licenses, _ := s.Store.ListLicenses()
	since, count := s.Now().Add(-24*time.Hour).Unix(), 0
	for _, license := range licenses {
		if license.OrderID == "" && license.CreatedAt >= since {
			count++
		}
	}
	return count
}

func (s *Server) sendKey(license store.License) error {
	if s.Mailer == nil || license.Email == "" {
		return apiError{http.StatusServiceUnavailable, "mail_off", "Email is not set up on the server."}
	}
	subject, text := mail.LicenseEmail(s.Config.ProductName, license.Key, s.Config.PublicURL, s.Config.SupportEmail)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err := s.Mailer.Send(ctx, license.Email, subject, text)
	if err != nil {
		s.Log.Event("mail_error", map[string]any{"step": "owner_send", "key": logging.MaskKey(license.Key), "error": err.Error()})
	}
	return err
}

func (s *Server) ownerNote(r *http.Request, call *ownerCall) (any, error) {
	request, err := readKeyRequest(r)
	if err != nil {
		return nil, err
	}
	license, err := s.Store.UpdateLicense(request.Key, func(l *store.License) error { l.Note = request.Note; return nil })
	s.ownerAudit("note", call.ip, logging.MaskKey(request.Key), err == nil)
	return license, err
}

func (s *Server) ownerRevoke(r *http.Request, call *ownerCall) (any, error) {
	request, err := readKeyRequest(r)
	if err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(request.Reason)
	if reason == "" {
		reason = "owner"
	}
	license, err := s.Store.Revoke(request.Key, reason)
	s.ownerAudit("revoke", call.ip, logging.MaskKey(request.Key)+" "+reason, err == nil)
	return license, err
}

func (s *Server) ownerResend(r *http.Request, call *ownerCall) (any, error) {
	request, err := readKeyRequest(r)
	if err != nil {
		return nil, err
	}
	license, err := s.Store.GetLicense(request.Key)
	if err != nil {
		return nil, err
	}
	if license.Revoked {
		return nil, store.ErrRevoked
	}
	if license.Email == "" {
		return nil, badRequest("This key has no email address.")
	}
	err = s.sendKey(license)
	s.ownerAudit("resend", call.ip, logging.MaskKey(request.Key), err == nil)
	if err != nil {
		return nil, apiError{http.StatusBadGateway, "mail_failed", "The email could not be sent. See Issues for the reason."}
	}
	return map[string]any{"emailed": license.Email}, nil
}

func (s *Server) ownerDelete(r *http.Request, call *ownerCall) (any, error) {
	request, err := readKeyRequest(r)
	if err != nil {
		return nil, err
	}
	license, err := s.Store.GetLicense(request.Key)
	if err != nil {
		return nil, err
	}
	if license.OrderID != "" {
		return nil, badRequest("Paid keys stay on record. Revoke it instead.")
	}
	err = s.Store.DeleteLicense(request.Key)
	s.ownerAudit("delete", call.ip, logging.MaskKey(request.Key), err == nil)
	return map[string]any{"deleted": request.Key}, err
}
