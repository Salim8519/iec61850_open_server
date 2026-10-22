package service

import (
	"errors"
	"math"

	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// The routing table of SCADA commands (ADR-0107): what the outstation decoded ("wire") says
// which step of a command it is, and the step and the point decide what ScadaBench asks the
// IED for. Pure functions; the executor carries out the answer.

// Phase is the step of a command: select, execute or cancel (deactivate).
type Phase string

const (
	PhaseSelect  Phase = "select"
	PhaseExecute Phase = "execute"
	PhaseCancel  Phase = "cancel"
)

// Action is what the executor does for one command.
type Action string

const (
	ActLegacy  Action = "legacy"  // the single/double command path of the Python engine (one MMS operate)
	ActSelect  Action = "select"  // SelectTyped
	ActOperate Action = "operate" // OperateTyped
	ActCancel  Action = "cancel"  // CancelTyped
)

// IEC 104 causes of transmission that start a command.
const (
	cotActivation   = 6
	cotDeactivation = 8
)

// PhaseOf maps the decoded request to a phase, or gives the plain reason it is refused. An
// event without "wire" (the Python-era channel, or a command typed on the page) executes.
func PhaseOf(protocol string, wire *pyjson.Object) (Phase, string) {
	if wire == nil {
		return PhaseExecute, ""
	}
	switch protocol {
	case "iec104":
		cot, _ := pytext.Int64(get(wire, "cot"))
		switch {
		case cot == cotDeactivation:
			return PhaseCancel, ""
		case cot != cotActivation:
			return "", "the cause of transmission must be 6 (activation) or 8 (deactivation)"
		case pytext.Truthy(get(wire, "select")):
			return PhaseSelect, ""
		}
		return PhaseExecute, ""
	case "dnp3":
		switch get(wire, "function") {
		case "select":
			return PhaseSelect, ""
		case "operate", "direct", "direct_no_ack":
			return PhaseExecute, ""
		}
		return "", "the DNP3 function must be select, operate, direct operate or direct operate without answer"
	}
	return PhaseExecute, "" // Modbus FC5/FC6/FC15/FC16 write at once
}

// Route is the action for a phase on a point of kind. An execute of a single or double
// command keeps the existing path unless a typed select of the IED came first.
func Route(kind string, phase Phase, typedSelected bool) Action {
	switch phase {
	case PhaseSelect:
		return ActSelect
	case PhaseCancel:
		return ActCancel
	}
	if gateway.IsTypedKind(kind) || typedSelected {
		return ActOperate
	}
	return ActLegacy
}

// NotAvailable reports the "not_available" answer of an engine without typed commands.
func NotAvailable(err error) bool {
	var coded *apperr.Error
	return errors.As(err, &coded) && coded.Code == "not_available"
}

// SelectFallback: a single or double command whose IED select is not available yet is
// confirmed by the gateway itself, as before ADR-0107; its execute then takes the legacy path.
func SelectFallback(kind string, err error) bool {
	return NotAvailable(err) && !gateway.IsTypedKind(kind)
}

// CommandValue is the IED value of a command and its words for the logs, or a plain reason
// why the value cannot be sent. raw is what the outstation decoded: a bool (single, double,
// CROB), a number (IEC 104 step state 1/2, a set point) or "higher"/"lower" from the page.
// A set point is divided by the point's scale (SCADA sees IED value × scale) and rounded for
// an INC control.
func CommandValue(kind string, raw any, scale float64, cdc string) (any, string, string) {
	switch kind {
	case gateway.KindStep:
		step := stepOf(raw)
		if step == "" {
			return nil, "invalid step", "the step must be lower (1) or higher / raise (2); 0 and 3 are not allowed"
		}
		return step, map[string]string{"higher": "raise", "lower": "lower"}[step], ""
	case gateway.KindSetpoint:
		number, ok := pytext.Float(raw)
		if _, isBool := raw.(bool); isBool || !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return nil, "invalid set point", "the set point must be a number"
		}
		if scale == 0 {
			scale = 1
		}
		value := number / scale
		words := "set point " + pytext.FormatG(number, 6)
		if cdc == "INC" {
			whole := math.Round(value)
			if whole < math.MinInt32 || whole > math.MaxInt32 {
				return nil, words, "the set point is outside the whole-number range of the INC control"
			}
			return int64(whole), words, ""
		}
		return value, words, ""
	}
	value := pytext.Truthy(raw)
	return value, map[bool]string{true: "ON / close", false: "OFF / open"}[value], ""
}

// stepOf reads a step: "higher"/"lower", true (close / ON = raise) or false, or the IEC 104
// regulating step state (1 lower, 2 higher). Anything else is "".
func stepOf(raw any) string {
	switch value := raw.(type) {
	case string:
		if value == "higher" || value == "lower" {
			return value
		}
		return ""
	case bool:
		return map[bool]string{true: "higher", false: "lower"}[value]
	}
	switch number, _ := pytext.Int64(raw); {
	case !pytext.IsInt(raw):
		return ""
	case number == 1:
		return "lower"
	case number == 2:
		return "higher"
	}
	return ""
}

// CommandInput reports a value the page may send to a point of kind: a bool, or (ADR-0107)
// "higher"/"lower" for a step and a finite number for a set point. Other points keep the
// Python rule (bool only).
func CommandInput(kind string, value any) bool {
	if _, isBool := value.(bool); isBool {
		return true
	}
	switch kind {
	case gateway.KindStep:
		return value == "higher" || value == "lower"
	case gateway.KindSetpoint:
		number, ok := pytext.Float(value)
		return ok && !math.IsNaN(number) && !math.IsInf(number, 0)
	}
	return false
}
