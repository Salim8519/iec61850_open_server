// Package typedcmd turns what a user asks for (a value and the command options) into the
// typed ctlVal the native control client sends (ADR-0104). It is pure: no I/O, no Flask
// shapes, shared by the runtime routes and IED Explorer.
package typedcmd

import (
	"math"
	"strconv"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Options are the command options of IEC 61850-7-2: test, check conditions and originator.
type Options struct {
	Test           bool   `json:"test"`
	InterlockCheck bool   `json:"interlock_check"`
	SynchroCheck   bool   `json:"synchro_check"`
	OrCat          int    `json:"orcat"`
	OrIdent        string `json:"orident"`
}

// DefaultOrCat is remote control (orCat 3), what a SCADA or engineering client sends.
const DefaultOrCat = 3

// Check refuses options a device would reject.
func (o Options) Check() error {
	if o.OrCat < 0 || o.OrCat > 8 {
		return apperr.ValueError("Originator category must be between 0 and 8 (3 is remote control)")
	}
	if len(o.OrIdent) > 64 {
		return apperr.ValueError("Originator name is at most 64 characters")
	}
	return nil
}

// Fields adds the options to a native request in the order studio_control reads them.
func (o Options) Fields(fields *pyjson.Object) *pyjson.Object {
	ident := o.OrIdent
	if ident == "" {
		ident = "ScadaBench"
	}
	return fields.Set("or_ident", ident).Set("or_cat", int64(o.OrCat)).
		Set("interlock_check", o.InterlockCheck).Set("synchro_check", o.SynchroCheck).Set("test", o.Test)
}

// Control is what the engine knows about one controllable object.
type Control struct {
	Ref       string
	CDC       string
	ValueKind string // bool, int, enum, float or step
	Labels    *pyjson.Object
	Min, Max  any
	Model     int64
	SboMs     int64
}

// Wire returns the native value kind and value for a user value. current is the feedback
// value now (ISC higher/lower moves it one position); it may be nil otherwise.
func Wire(control Control, value any, current any) (string, any, error) {
	switch control.ValueKind {
	case "bool":
		if flag, ok := value.(bool); ok {
			return "bool", flag, nil
		}
		return "", nil, apperr.ValueError("Control value must be boolean") // the old route's text
	case "enum":
		return enumValue(control, value)
	case "step":
		if text, ok := value.(string); ok && (text == "higher" || text == "lower" || text == "stop") {
			return "step", text, nil
		}
		return "", nil, apperr.ValueError("This command moves one step: send higher, lower or stop")
	case "float":
		number, ok := asFloat(value)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) {
			return "", nil, apperr.ValueError("This command needs a number")
		}
		if err := inLimits(control, number); err != nil {
			return "", nil, err
		}
		return "float", number, nil
	case "int":
		if text, ok := value.(string); ok && (text == "higher" || text == "lower") {
			now, known := asFloat(current)
			if !known {
				return "", nil, apperr.ValueError("The present position is unknown; type the position to go to")
			}
			if text == "higher" {
				value = int64(now) + 1
			} else {
				value = int64(now) - 1
			}
		}
		number, ok := asFloat(value)
		if !ok || number != math.Trunc(number) {
			return "", nil, apperr.ValueError("This command needs a whole number")
		}
		if err := inLimits(control, number); err != nil {
			return "", nil, err
		}
		return "int", int64(number), nil
	}
	return "", nil, apperr.ValueError(control.Ref + " cannot be commanded by ScadaBench")
}

func enumValue(control Control, value any) (string, any, error) {
	if control.Labels != nil {
		if text, ok := value.(string); ok {
			if ord, found := control.Labels.Get(text); found {
				return "enum", ord, nil
			}
		}
		if number, ok := asFloat(value); ok {
			for _, label := range control.Labels.Keys() {
				if ord, _ := control.Labels.Get(label); ordEquals(ord, number) {
					return "enum", int64(number), nil
				}
			}
		}
	}
	return "", nil, apperr.ValueError("Pick one of the listed values for " + control.Ref)
}

func ordEquals(ord any, number float64) bool {
	value, ok := asFloat(ord)
	return ok && value == number
}

func inLimits(control Control, number float64) error {
	low, okLow := asFloat(control.Min)
	high, okHigh := asFloat(control.Max)
	if okLow && okHigh && (number < low || number > high) {
		return apperr.ValueError("The value must be between " + format(low) + " and " + format(high))
	}
	return nil
}

func format(number float64) string { return strconv.FormatFloat(number, 'f', -1, 64) }

func asFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case int64:
		return float64(v), true
	case int:
		return float64(v), true
	case float64:
		return v, true
	case float32:
		return float64(v), true
	}
	return 0, false
}

// ModelName is the short control model name the routes answer with.
func ModelName(model int64) string {
	switch model {
	case 1:
		return "direct-normal"
	case 2:
		return "sbo-normal"
	case 3:
		return "direct-enhanced"
	case 4:
		return "sbo-enhanced"
	}
	return "status-only"
}
