package acquisition

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
)

// ReadOne reads one point of the device now, read only, applies it as a live value (so every
// page sees it) and returns the stored value. Devices uses it to observe a command's feedback
// (ADR-0138); found is false when the device did not give a value.
func (s *RealSession) ReadOne(ref, fc string) (value any, found bool, err error) {
	if _, err := s.Read([]any{pyv.O("ref", ref, "fc", fc)}, "live_poll"); err != nil {
		return nil, false, err
	}
	if s.deps.Values == nil {
		return nil, false, apperr.RuntimeError("live values are not available")
	}
	value, _, found = s.deps.Values.Get(s.ID(), ref)
	return value, found, nil
}

// State is the session state word (starting, connecting, discovering, live, retrying, ...).
func (s *RealSession) State() string { return pyv.GetStr(s.Status(), "state") }
