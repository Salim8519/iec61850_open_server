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
// license. It returns the server's verdict (any decoded JSON value), or nil.
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
	client, err := s.client()
	if err != nil {
		return nil, err
	}
	answer, err := client.Verify(status.LicenseID, bound)
	var server *ServerError
	if errors.As(err, &server) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.applyVerdict(answerObject(answer), bound)
}

// applyVerdict stores what the server said: a revoked or moved license loses its
// certificate (renamed, never deleted); a valid one may receive a fresh certificate.
func (s *Service) applyVerdict(object *pyjson.Object, bound string) (any, error) {
	if object == nil {
		return nil, uncaught("license server answer is not an object")
	}
	verdict, _ := object.Get("status")
	switch verdict.(type) {
	case []any, *pyjson.Object:
		return nil, uncaught("unhashable verdict")
	}
	update := pyjson.NewObject().Set("last_verify", s.now())
	text, _ := verdict.(string)
	switch {
	case text == "revoked" || text == "moved":
		reason := MovedText
		if text == "revoked" {
			reason = RevokedText
		}
		update.Set("revoked_reason", reason)
		target := filepath.Join(filepath.Dir(s.certFile), fmt.Sprintf("license.cert.%s-%d", text, s.now()))
		if err := os.Rename(s.certFile, target); err != nil {
			return nil, err
		}
		s.invalidate()
	case text == "valid":
		if err := s.refreshCertificate(object, bound); err != nil {
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
