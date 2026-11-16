package realcmd

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
)

// Session is the live connection of one real IED (acquisition.RealSession).
type Session interface {
	State() string
	ReadOne(ref, fc string) (value any, found bool, err error)
}

// Service sends commands for Devices. Sessions answers nil, error when the IED has none.
type Service struct {
	FindIED  func(key string) (*pyjson.Object, error)
	Sessions func(iedID string) (Session, error)
	Control  control.Operator
	Emit     func(kind string, payload *pyjson.Object)
	Observe  time.Duration // how long the feedback may take; 3 s when zero
	Poll     time.Duration // between feedback reads; 500 ms when zero
}

// Operate sends one confirmed command and returns {ok, summary, result, observed, changed,
// correlation_id}. A refusal is a ValueError in words; the device's own refusal is ok false.
func (s *Service) Operate(key, ref string, value any, confirmed bool) (*pyjson.Object, error) {
	ied, err := s.FindIED(key)
	if err != nil {
		return nil, err
	}
	iedID := text(ied, "id")
	var session Session
	state := ""
	if found, err := s.Sessions(iedID); err == nil && found != nil {
		session, state = found, found.State()
	}
	if reason := Refusal(Request{IED: ied, Ref: ref, Value: value, Confirmed: confirmed, State: state}); reason != "" {
		return nil, apperr.ValueError(reason)
	}
	if s.Control == nil {
		return nil, apperr.RuntimeError("Commands are not available in this copy of ScadaBench")
	}
	command := controlOf(ied, ref)
	host, port, _ := netset.RealEndpoint(ied)
	correlation := newID()
	s.Emit("real_command_requested", pyjson.NewObject().Set("ied_id", iedID).Set("ref", ref).Set("value", value).
		Set("correlation_id", correlation).Set("origin", "devices"))
	result, err := s.Control.Operate(hostText(host), port, ref, value, pyjson.NewObject().Set("originator", "station"))
	if err != nil {
		s.Emit("real_command_failed", pyjson.NewObject().Set("ied_id", iedID).Set("ref", ref).
			Set("correlation_id", correlation).Set("reason", err.Error()))
		return nil, err
	}
	summary := control.Summary(result)
	ok, _ := get(result, "ok").(bool)
	kind := map[bool]string{true: "real_command_applied", false: "real_command_failed"}[ok]
	s.Emit(kind, pyjson.NewObject().Set("ied_id", iedID).Set("ref", ref).Set("correlation_id", correlation).
		Set("applied", summary))
	answer := pyjson.NewObject().Set("ok", ok).Set("summary", summary).Set("result", result).
		Set("observed", nil).Set("changed", false).Set("correlation_id", correlation)
	if !ok {
		return answer, nil
	}
	observed, changed := s.observe(session, text(command, "status"), expected(command, value.(bool)))
	s.Emit("real_command_observed", pyjson.NewObject().Set("ied_id", iedID).Set("ref", ref).
		Set("correlation_id", correlation).Set("observed", observed).Set("changed", changed))
	return answer.Set("observed", observed).Set("changed", changed), nil
}

// observe reads the feedback point until it shows want or the time is up; it says what it saw.
func (s *Service) observe(session Session, status string, want any) (string, bool) {
	limit, pause := s.Observe, s.Poll
	if limit == 0 {
		limit = 3 * time.Second
	}
	if pause == 0 {
		pause = 500 * time.Millisecond
	}
	deadline := time.Now().Add(limit)
	seen := "not read"
	for {
		value, found, err := session.ReadOne(status, "ST")
		if err == nil && found {
			seen = positionText(value)
			if value == want {
				return status + " is " + seen, true
			}
		}
		if time.Now().After(deadline) {
			return status + " still " + seen + " after " + limit.String(), false
		}
		time.Sleep(pause)
	}
}

func hostText(host any) string { value, _ := host.(string); return value }

func newID() string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	return hex.EncodeToString(raw[:])
}
