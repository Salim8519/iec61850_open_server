package api

import (
	"errors"
	"net/http"
	"strings"

	"substation-license-server/internal/certs"
	"substation-license-server/internal/store"
)

// verify lets installed apps learn about refunds and revocations. Every answer keeps the
// fields old apps read (status, certificate, server_time) and adds "answer": the status,
// machine, key and the request's nonce signed with the certificate key, so a new app can
// prove the reply is genuine and belongs to its own request.
func (s *Server) verify(r *http.Request, ip string) (any, error) {
	request, err := s.readMachine(r, false, false)
	if err != nil {
		return nil, err
	}
	if request.Nonce != "" && !certs.ValidNonce(request.Nonce) {
		return nil, badRequest("nonce must be 32 lowercase hex characters")
	}
	if len(request.SentKey) > maxKeyEcho {
		return nil, badRequest("license_key is too long")
	}
	if request.LicenseKey == "" || strings.HasPrefix(request.LicenseKey, "TRIAL-") {
		return s.verifyReply(request, certs.StatusValid, nil), nil
	}
	license, err := s.Store.GetLicense(request.LicenseKey)
	if errors.Is(err, store.ErrUnknownKey) {
		return s.verifyReply(request, certs.StatusUnknown, nil), nil
	}
	if err != nil {
		return nil, err
	}
	switch {
	case license.Revoked:
		return s.verifyReply(request, certs.StatusRevoked, nil), nil
	case license.HWID != request.HWID:
		return s.verifyReply(request, certs.StatusMoved, nil), nil
	}
	bundle, err := s.CertificateFor(license, request.HWID)
	return s.verifyReply(request, certs.StatusValid, &bundle), err
}

// maxKeyEcho bounds the license_key the signed answer repeats; real keys are under 30 characters.
const maxKeyEcho = 128

// verifyReply builds one verify answer: the old fields, the signed server time and the
// signed answer, all stamped with the same second. A signing failure leaves the signed
// parts out (logged); old apps and new apps both handle a missing signature.
func (s *Server) verifyReply(request machineRequest, status string, bundle *certs.Bundle) map[string]any {
	now := s.Now().Unix()
	reply := map[string]any{"status": status}
	if bundle != nil {
		reply["certificate"] = *bundle
	}
	s.withTimeAt(reply, now)
	signed, err := s.Signer.SignAnswer(certs.Answer{HWID: request.HWID, LicenseID: request.SentKey,
		Nonce: request.Nonce, Status: status, Time: now})
	if err != nil {
		s.Log.Event("verify_answer_error", map[string]any{"error": err.Error()})
		return reply
	}
	reply["answer"] = signed
	return reply
}
