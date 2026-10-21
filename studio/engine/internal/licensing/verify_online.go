package licensing

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"

	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// Revocation texts stored as license_state.json "revoked_reason".
const (
	RevokedText = "This license was cancelled. Contact the seller if this is a mistake."
	MovedText   = "This license key was moved to another computer."
)

// number is float(value) for the int/float last_verify; ok is false for the TypeError
// Python raises on other types.
func number(value any) (float64, bool) {
	switch v := value.(type) {
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case *big.Int:
		f, _ := new(big.Float).SetInt(v).Float64()
		return f, true
	case float64:
		return v, true
	case bool:
		if v {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// verifyLifetime is OnlineChecks._verify_lifetime: once every VerifyEverySeconds a
// lifetime license learns about refunds or key moves; offline PCs simply keep their
// license. Only a genuine signed answer counts (signedVerdict); anything else is treated
// like an unreachable server. It returns the signed verdict, or nil.
func (s *Service) verifyLifetime(status Status) (any, error) {
	stateValue, readable := pytext.ReadJSON(s.stateFile)
	if !status.Licensed || status.Plan != "lifetime" {
		return nil, nil
	}
	state, isObject := stateValue.(*pyjson.Object)
	if !readable {
		state, isObject = pyjson.NewObject(), true
	}
	if !isObject {
		return nil, uncaught("'license_state.json' does not hold a JSON object")
	}
	lastVerify, found := state.Get("last_verify")
	if !found {
		lastVerify = int64(0)
	}
	last, ok := number(lastVerify)
	if !ok {
		return nil, uncaught("last_verify is not a number")
	}
	if s.options.Clock()-last < VerifyEverySeconds {
		return nil, nil
	}
	s.mu.Lock()
	bound := s.boundHWID
	s.mu.Unlock()
	var err error
	if bound == "" {
		if bound, err = s.HWID(); err != nil {
			return nil, err
		}
	}
	verdict, answer, err := s.signedVerdict(status.LicenseID, bound)
	var server *ServerError
	if errors.As(err, &server) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.applyVerdict(status.LicenseID, verdict, answer, bound)
}

// signedVerdict is OnlineChecks._signed_verdict: ask the server about a license with a
// fresh nonce; the verdict and the whole answer come back only for a genuine signed answer,
// otherwise a "bad_answer" ServerError (which callers treat like an unreachable server).
func (s *Service) signedVerdict(licenseID any, machine string) (string, *pyjson.Object, error) {
	nonce := s.options.Nonce()
	client, err := s.client()
	if err != nil {
		return "", nil, err
	}
	answer, err := client.Verify(licenseID, machine, nonce)
	if err != nil {
		return "", nil, err
	}
	verdict, ok := VerifiedStatus(answer, nonce, licenseID, machine, s.options.Keys)
	if !ok {
		return "", nil, &ServerError{Message: BadAnswerText, Code: "bad_answer"}
	}
	return verdict, answerObject(answer), nil
}

// cancel is OnlineChecks._cancel: the server cancelled this license, so remember it for
// good in all three places, then keep the certificate file aside (renamed, never deleted).
// It returns the reason for license_state.json.
func (s *Service) cancel(licenseID any, verdict string) string {
	s.filesMu.Lock()
	defer s.filesMu.Unlock()
	s.revoked.Add(licenseID)
	target := filepath.Join(filepath.Dir(s.certFile), fmt.Sprintf("license.cert.%s-%d", verdict, s.now()))
	_ = os.Rename(s.certFile, target) // the remembered ID already refuses the certificate
	s.invalidate()
	if verdict == "revoked" {
		return RevokedText
	}
	return MovedText
}

// applyVerdict stores what the server said: a cancelled license is remembered and loses
// its certificate; a valid one may receive a fresh certificate.
func (s *Service) applyVerdict(licenseID any, verdict string, answer *pyjson.Object, bound string) (any, error) {
	update := pyjson.NewObject().Set("last_verify", s.now())
	switch verdict {
	case "revoked", "moved":
		update.Set("revoked_reason", s.cancel(licenseID, verdict))
	case "valid":
		if err := s.refreshCertificate(answer, bound); err != nil {
			return nil, err
		}
	}
	if err := s.updateState(update); err != nil {
		return nil, err
	}
	return verdict, nil
}

// updateState re-reads license_state.json and merges update into it, as Python's
// state.update() under the lock (other keys, such as the clock copies, are kept).
func (s *Service) updateState(update *pyjson.Object) error {
	s.filesMu.Lock()
	defer s.filesMu.Unlock()
	state, err := readObject(s.stateFile)
	if err != nil {
		return err
	}
	for _, key := range update.Keys() {
		value, _ := update.Get(key)
		state.Set(key, value)
	}
	return s.writeState(state)
}

// refreshCertificate keeps a reissued certificate for this PC; a legacy-bound certificate
// is kept as is (a reissued one would no longer qualify for the legacy machine ID).
func (s *Service) refreshCertificate(answer *pyjson.Object, bound string) error {
	fresh, found := answer.Get("certificate")
	if !found || !truthy(fresh) {
		return nil
	}
	hwid, err := s.HWID()
	if err != nil || bound != hwid {
		return err
	}
	bundle, err := ParseBundle(fresh)
	var bad *CertificateError
	if errors.As(err, &bad) {
		return nil
	}
	if err != nil {
		return err
	}
	payload, err := VerifyBundle(bundle, s.options.Keys)
	if errors.As(err, &bad) {
		return nil
	}
	if err != nil {
		return err
	}
	if certificateHWID, _ := payload.Get("hwid"); certificateHWID != any(hwid) {
		return nil
	}
	text, err := pytext.DumpsIndent(bundle.Object())
	if err != nil {
		return err
	}
	s.filesMu.Lock()
	defer s.filesMu.Unlock()
	return pytext.WriteReplace(s.certFile, text)
}
