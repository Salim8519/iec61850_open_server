package realcmd

import (
	"errors"

	"studio-engine/internal/apperr"
	"studio-engine/internal/project/netset"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/control"
	"studio-engine/internal/tools/worker"
	"studio-engine/internal/typedcmd"
)

// Step is one command step from Devices (ADR-0149): Select, Operate or Cancel of any command the
// CID gives (SPC, DPC, INC, ENC, APC, BAC, BSC, ISC), with the IEC 61850 command options.
type Step struct {
	Action    string // select, operate or cancel
	Ref       string
	Value     any // true/false, a number, an ENC label, or "higher"/"lower" for a step command
	Options   typedcmd.Options
	Confirmed bool
	// Selected: the person pressed Select first (step by step), so an Operate of a select-before-
	// operate control uses that selection. Otherwise Operate selects and operates in one go.
	Selected bool
}

// directWindowMs is how long Operate is offered after a Select of a direct control.
const directWindowMs = 30000

var stepOps = map[string]string{"select": "typed_select", "operate": "typed_operate", "cancel": "typed_cancel"}

// Command sends one confirmed step and returns {ok, action, summary, model, expires_ms, result,
// observed, changed, correlation_id}. A reason not to send is a ValueError in words; the device's
// own refusal is ok false with a plain-English summary.
func (s *Service) Command(key string, step Step) (*pyjson.Object, error) {
	ied, err := s.FindIED(key)
	if err != nil {
		return nil, err
	}
	iedID := text(ied, "id")
	session, state := s.session(iedID)
	target, status := s.commandOf(ied, step.Ref)
	if reason := StepRefusal(StepRequest{IED: ied, Step: step, Known: target.Ref != "", State: state}); reason != "" {
		return nil, apperr.ValueError(reason)
	}
	if err := step.Options.Check(); err != nil {
		return nil, err
	}
	operator, typed := s.Control.(control.TypedOperator)
	if s.Control == nil || !typed {
		return nil, apperr.RuntimeError("Commands are not available in this copy of ScadaBench")
	}
	fields, before, err := s.stepFields(ied, target, status, session, step)
	if err != nil {
		return nil, err
	}
	if !s.begin(iedID, step.Ref) {
		return nil, apperr.ValueError("A command to " + step.Ref + " on " + text(ied, "name") +
			" is still running; wait for its result")
	}
	defer s.end(iedID, step.Ref)
	correlation := newID()
	s.Emit("real_command_requested", pyjson.NewObject().Set("ied_id", iedID).Set("ref", step.Ref).
		Set("action", step.Action).Set("value", step.Value).Set("test", step.Options.Test).
		Set("correlation_id", correlation).Set("origin", "devices"))
	result, err := operator.Typed(stepOps[step.Action], fields)
	if err != nil {
		return nil, s.stepError(ied, step.Ref, correlation, err)
	}
	ok, _ := get(result, "ok").(bool)
	summary := stepSummary(step.Action, result)
	if !ok {
		var behaviour any
		if session != nil {
			behaviour, _, _ = session.ReadOne(behaviourRef(step.Ref), "ST")
		}
		summary = stepWord(step.Action) + " refused by " + text(ied, "name") + ": " +
			Explain(result, behaviour, step.Options.Test)
	}
	s.Emit(stepEvent(step.Action, ok), pyjson.NewObject().Set("ied_id", iedID).Set("ref", step.Ref).
		Set("correlation_id", correlation).Set("applied", summary))
	window := 0
	if expires, _ := get(result, "expires_ms").(int64); ok && step.Action == "select" {
		if selected, _ := get(result, "selected").(bool); !selected {
			expires = directWindowMs // nothing is reserved: the Operate goes straight through
		}
		window = typedcmd.OperateWindow(int(expires))
	}
	answer := pyjson.NewObject().Set("ok", ok).Set("action", step.Action).Set("summary", summary).
		Set("model", get(result, "model")).Set("expires_ms", int64(window)).Set("result", result).
		Set("observed", nil).Set("changed", false).Set("correlation_id", correlation)
	if !ok || step.Action != "operate" || session == nil || status == "" {
		return answer, nil
	}
	observed, changed := s.observeStep(session, target, status, step.Value, before)
	s.Emit("real_command_observed", pyjson.NewObject().Set("ied_id", iedID).Set("ref", step.Ref).
		Set("correlation_id", correlation).Set("observed", observed).Set("changed", changed))
	return answer.Set("observed", observed).Set("changed", changed), nil
}

// stepFields is the native request: endpoint, card, options and the typed value. before is the
// feedback read before sending (a step command moves from it; nil when not read).
func (s *Service) stepFields(ied *pyjson.Object, target typedcmd.Control, status string, session Session,
	step Step) (*pyjson.Object, any, error) {
	options := step.Options
	if options.OrCat == 0 {
		options.OrCat = typedcmd.DefaultOrCat
	}
	host, port, _ := netset.RealEndpoint(ied)
	fields := options.Fields(pyjson.NewObject().Set("ip", hostText(host)).Set("port", port).Set("ref", step.Ref))
	if local := localCard(ied); local != "" {
		fields.Set("local", local) // through the card chosen for this relay (change 0233)
	}
	var before any
	if session != nil && status != "" && step.Action == "operate" {
		before, _, _ = session.ReadOne(status, feedbackFC(status))
	}
	if step.Action == "cancel" || (step.Action == "select" && step.Value == nil) {
		return fields, before, nil
	}
	kind, wire, err := typedcmd.Wire(target, step.Value, before)
	if err != nil {
		return nil, nil, err
	}
	fields.Set("value", wire).Set("kind", kind)
	if step.Action == "operate" {
		fields.Set("require_select", step.Selected)
	}
	return fields, before, nil
}

// stepError records a failed step; with no answer in time the result is unknown.
func (s *Service) stepError(ied *pyjson.Object, ref, correlation string, err error) error {
	iedID := text(ied, "id")
	var timeout *worker.Timeout
	if errors.As(err, &timeout) {
		reason := text(ied, "name") + " did not answer in time; result unknown — check the position before trying again"
		s.Emit("real_command_unknown", pyjson.NewObject().Set("ied_id", iedID).Set("ref", ref).
			Set("correlation_id", correlation).Set("reason", reason))
		return apperr.RuntimeError(reason)
	}
	s.Emit("real_command_failed", pyjson.NewObject().Set("ied_id", iedID).Set("ref", ref).
		Set("correlation_id", correlation).Set("reason", err.Error()))
	return err
}

func (s *Service) session(iedID string) (Session, string) {
	if found, err := s.Sessions(iedID); err == nil && found != nil {
		return found, found.State()
	}
	return nil, ""
}

func localCard(ied *pyjson.Object) string {
	settings, _ := get(ied, "live").(*pyjson.Object)
	local, _ := get(settings, "local").(string)
	return local
}
