package updates

import (
	"errors"
	"math"
	"math/big"

	"studio-engine/internal/licensing"
	"studio-engine/internal/licensing/pytext"
	"studio-engine/internal/pyjson"
)

// checkedAt is `state.get("checked_at") or 0` used as a number; another type is the
// TypeError Python raises (HTTP 500 for "Check now", a logged failure in the background).
func checkedAt(state *pyjson.Object) (float64, error) {
	switch value := get(state, "checked_at").(type) {
	case nil:
		return 0, nil
	case bool:
		if value {
			return 1, nil
		}
		return 0, nil
	case int64:
		return float64(value), nil
	case *big.Int:
		f, _ := new(big.Float).SetInt(value).Float64()
		return f, nil
	case float64:
		return value, nil
	case string, []any, *pyjson.Object:
		if !licensing.Truthy(value) {
			return 0, nil
		}
	}
	return 0, &licensing.UncaughtError{Reason: "unsupported operand type for checked_at"}
}

// Due is UpdateService.due: auto on and a day since the last check.
func (s *Service) Due() (bool, error) {
	state := s.read()
	if auto, found := state.Get("auto"); found && auto == any(false) {
		return false, nil
	}
	checked, err := checkedAt(state)
	if err != nil {
		return false, err
	}
	return s.options.Clock()-checked >= Day, nil
}

// fetchError is the ConnectionError of _fetch; its text is the stored error code.
type fetchError struct{ code string }

func (e *fetchError) Error() string { return e.code }

// Check is UpdateService.check: ask the server now. A failure is stored quietly as an
// error code ("offline", "no_server", "no_release", "server_error", "bad_answer"), never
// returned; manual checks within ManualGap reuse the last answer. An error result stands
// for an exception Python would not catch (the state is still written first).
func (s *Service) Check(manual bool) (*pyjson.Object, error) {
	s.mu.Lock()
	state := s.read()
	if manual {
		checked, err := checkedAt(state)
		if err != nil {
			s.mu.Unlock()
			return nil, err
		}
		if s.options.Clock()-checked < ManualGap {
			s.mu.Unlock()
			return s.Status()
		}
	}
	s.checking.Store(true)
	s.mu.Unlock()
	uncaught := s.ask(state)
	s.mu.Lock()
	s.checking.Store(false)
	merged := s.read()
	for _, key := range []string{"release", "etag", "error"} {
		merged.Set(key, get(state, key))
	}
	merged.Set("checked_at", int64(math.Trunc(s.options.Clock())))
	err := s.write(merged)
	s.mu.Unlock()
	if uncaught != nil {
		return nil, uncaught
	}
	if err != nil {
		return nil, err
	}
	return s.Status()
}

// ask fetches and verifies, updating state in place as the try/except of check() does;
// it returns only errors Python does not catch.
func (s *Service) ask(state *pyjson.Object) error {
	etag, _ := state.Get("etag")
	raw, newTag, err := s.fetch(etag)
	var connection *fetchError
	switch {
	case errors.As(err, &connection):
		state.Set("error", connection.code)
		return nil
	case err != nil:
		return err
	}
	if raw != nil {
		record, err := s.verify(raw)
		if errors.Is(err, errBadRecord) {
			state.Set("error", "bad_answer")
			return nil
		}
		if err != nil {
			return err
		}
		state.Set("release", record).Set("etag", newTag)
	}
	state.Set("error", nil)
	return nil
}

// verify is UpdateService._verify: size limit, JSON envelope, Ed25519 over
// ReleaseDomain + payload with the kid's key (or every key), then CleanRecord.
func (s *Service) verify(raw []byte) (*pyjson.Object, error) {
	if len(raw) > MaxBody {
		return nil, errBadRecord
	}
	decoded, err := pytext.LoadsBytes(raw)
	if err != nil {
		return nil, errBadRecord
	}
	body, isObject := decoded.(*pyjson.Object)
	if !isObject {
		return nil, &licensing.UncaughtError{Reason: "release answer is not an object"}
	}
	payload, isText := get(body, "payload").(string)
	signature, isSignature := get(body, "signature").(string)
	if !isText || !isSignature {
		return nil, errBadRecord
	}
	signatureBytes, err := licensing.URLSafeB64Decode(licensing.PadBase64(signature))
	if err != nil {
		return nil, errBadRecord
	}
	candidates, err := s.candidates(get(body, "kid"))
	if err != nil {
		return nil, err
	}
	message := append([]byte(ReleaseDomain), payload...)
	for _, key := range candidates {
		if licensing.Verify(key, message, signatureBytes) {
			record, err := pytext.LoadsString(payload)
			if err != nil {
				return nil, errBadRecord
			}
			return CleanRecord(record)
		}
	}
	return nil, errBadRecord
}

// candidates is `[keys[kid]] if body.get("kid") in keys else list(keys.values())`; a list
// or object kid is unhashable (TypeError), which check() reports as "bad_answer".
func (s *Service) candidates(kid any) ([][]byte, error) {
	switch value := kid.(type) {
	case []any, *pyjson.Object:
		return nil, errBadRecord
	case string:
		if key, found := s.options.Keys[value]; found {
			return [][]byte{key}, nil
		}
	}
	all := make([][]byte, 0, len(s.options.Keys))
	for _, key := range s.options.Keys {
		all = append(all, key)
	}
	return all, nil
}
