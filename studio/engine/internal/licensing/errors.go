package licensing

import (
	"errors"

	"studio-engine/internal/pyjson"
)

// ActionError is service.LicenseActionError: a refused license action with its code
// ("invalid", "development", "bad_key", "bad_url", "not_valid_here", "bad_certificate",
// "bad_email").
type ActionError struct {
	Message string
	Code    string
}

func (e *ActionError) Error() string { return e.Message }

// UncaughtError stands for a Python exception that no handler catches (KeyError on a
// server answer without "certificate", TypeError on a list "kid", ...): Flask answers
// with its plain 500 page, so the HTTP layer must too.
type UncaughtError struct{ Reason string }

func (e *UncaughtError) Error() string { return e.Reason }

func uncaught(reason string) error { return &UncaughtError{Reason: reason} }

// serverCodes and conflictCodes are api/license.py's status rules.
var (
	serverCodes   = map[string]bool{"offline": true, "no_server": true, "server_error": true, "bad_time": true}
	conflictCodes = map[string]bool{"bound_elsewhere": true, "transfer_limit": true, "trial_used": true,
		"email_required": true, "email_used": true}
)

// ErrorResponse maps an action error as api/license.py `_run` does: ServerError 503
// (offline, no_server, server_error, bad_time), 409 (bound_elsewhere, transfer_limit,
// trial_used, email_required, email_used) or 400; ActionError and CertificateError 400; body {"error", "code"} for pyjson.Flask.
// ok is false for any other error (an UncaughtError or I/O failure: HTTP 500).
func ErrorResponse(err error) (status int, body *pyjson.Object, ok bool) {
	var server *ServerError
	var action *ActionError
	var certificate *CertificateError
	switch {
	case errors.As(err, &server):
		code, _ := server.Code.(string)
		status = 400
		if serverCodes[code] {
			status = 503
		} else if conflictCodes[code] {
			status = 409
		}
		return status, pyjson.NewObject().Set("error", server.Message).Set("code", server.Code), true
	case errors.As(err, &action):
		return 400, pyjson.NewObject().Set("error", action.Message).Set("code", action.Code), true
	case errors.As(err, &certificate):
		return 400, pyjson.NewObject().Set("error", certificate.Message).Set("code", "bad_certificate"), true
	}
	return 0, nil, false
}
