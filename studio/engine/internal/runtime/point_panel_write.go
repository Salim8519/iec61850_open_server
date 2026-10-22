package runtime

import (
	"math"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// pointWrite checks one status, double or value point against the running simulated IED and
// turns the typed value into the item the sample writer takes.
func pointWrite(target *running, point panelPoint, value any) (item, error) {
	info, found := target.model.lookup(point.ref)
	if !found {
		return item{}, apperr.New(point.label + " uses a signal this IED does not have: " + point.ref)
	}
	if !info.Writable || info.FC == "CF" {
		return item{}, apperr.New(point.label + " cannot be changed from here: its signal is read-only in the simulated IED.")
	}
	wire, err := pointWireValue(point, info.Kind, value)
	if err != nil {
		return item{}, err
	}
	return item{ref: point.ref, value: wire}, nil
}

// pointWireValue is the value the IED holds for what the browser sent: a status is true or
// false, a double point a Dbpos position 0 to 3, a value a number (whole for integer signals).
func pointWireValue(point panelPoint, signalKind string, value any) (any, error) {
	switch point.kind {
	case "status":
		if flag, isBool := value.(bool); isBool {
			return flag, nil
		}
		return nil, apperr.New(point.label + " is a status point: set it ON or OFF (true or false).")
	case "double":
		if number, ok := wholeNumber(value); ok && number >= 0 && number <= 3 {
			return number, nil
		}
		return nil, apperr.New(point.label + " is a double point: set it to 0 (intermediate), 1 (open), " +
			"2 (closed) or 3 (bad).")
	}
	number, isNumber := pyval.Float(value)
	if pyval.IsBool(value) || !isNumber || math.IsNaN(number) || math.IsInf(number, 0) {
		return nil, apperr.New(point.label + " is a value point: enter a number.")
	}
	if len(signalKind) >= 3 && signalKind[:3] == "INT" {
		rounded := math.Round(number)
		if math.Abs(number-rounded) > 1e-6*math.Max(1, math.Abs(number)) {
			return nil, apperr.New(point.label + " holds whole numbers, but " + pyval.Repr(number) +
				" is not whole. Check the scale of the point.")
		}
		return int64(rounded), nil
	}
	return number, nil
}

// wholeNumber reads an integer-valued JSON number.
func wholeNumber(value any) (int64, bool) {
	switch typed := value.(type) {
	case int64:
		return typed, true
	case float64:
		return int64(typed), typed == math.Trunc(typed) && math.Abs(typed) < 1e15
	}
	return 0, false
}

// pointSet writes one status, double or value point into the simulated IED, as a latching
// push button does; no command is sent.
func (c *Coordinator) pointSet(ied *pyjson.Object, point panelPoint, value any) (*pyjson.Object, error) {
	target, err := c.panelTarget(ied)
	if err != nil {
		return nil, err
	}
	write, err := pointWrite(target, point, value)
	if err != nil {
		return nil, err
	}
	result, err := c.apply(pyval.Str(ied, "id"), []item{write}, "equipment_button")
	if err != nil {
		return nil, err
	}
	actual, ok := result.Get(point.ref)
	if !ok {
		return nil, apperr.KeyError(point.ref)
	}
	return pyval.Obj("ref", point.ref, "value", actual), nil
}

// pointSend sends one command point through the canvas command path (Operate): the same MMS
// client, the same refusal of real devices, the command marked as ScadaBench's own. A Boolean
// SPC/DPC without a held selection keeps exactly that path; typed values and selected rows go
// through the typed Operate (ADR-0108).
func (c *Coordinator) pointSend(nodeID string, ied *pyjson.Object, point panelPoint, value any) (*pyjson.Object, error) {
	control := typedControl(ied, point.ref)
	if control != nil && !typedCDCs[pyval.Str(control, "cdc")] {
		control = nil
	}
	if control == nil {
		if _, isBool := value.(bool); !isBool {
			return nil, apperr.New(point.label + " is sent as true or false: on or off for a single command, " +
				"close or open for a double command.")
		}
	}
	if refusal := commandRefusal(ied); refusal != "" {
		return nil, apperr.ValueError(refusal)
	}
	if control == nil {
		return nil, apperr.New(point.label + " uses a command this IED does not have: " + point.ref)
	}
	typed, err := typedValue(control, value, point.label)
	if err != nil {
		return nil, err
	}
	iedID := pyval.Str(ied, "id")
	c.commandMu.Lock()
	defer c.commandMu.Unlock()
	c.access.markOwn(iedID, point.ref)
	c.mu.Lock()
	run, state := c.runs[iedID], c.iedStates[iedID]
	c.mu.Unlock()
	if run == nil || run.worker == nil {
		return nil, apperr.ValueError("IED is not running")
	}
	answer, err := c.sendPoint(nodeID, ied, point, control, typed, state, run)
	if err != nil {
		return nil, err
	}
	return pyval.Obj("ref", point.ref, "value", typed, "command", answer), nil
}
