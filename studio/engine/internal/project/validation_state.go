package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// validateSignalBinding checks one state binding: optional expected kind, then a
// runtime-editable (writable, not CF) signal.
func validateSignalBinding(binding *pyjson.Object, ieds *Lookup, expected string) (*pyjson.Object, error) {
	signal, err := bindingSignal(binding, ieds)
	if err != nil {
		return nil, err
	}
	if expected != "" && text(signal, "kind") != expected {
		return nil, apperr.New("State mapping requires a " + expected + " signal")
	}
	if !pyval.Truthy(pyval.Lookup(signal, "writable", nil)) || text(signal, "fc") == "CF" {
		return nil, apperr.New("State mapping requires a runtime-editable status signal")
	}
	return signal, nil
}

func validateState(data *pyjson.Object, kind string, ieds *Lookup) error {
	raw := pyval.Lookup(data, "state_mapping", nil)
	if !pyval.Truthy(raw) {
		return nil
	}
	mapping := object(raw)
	mode := text(mapping, "mode")
	source := object(pyval.Lookup(mapping, "source", nil))
	if name, found := drawingOnly[kind]; found {
		return apperr.New(name + " is a drawing point and does not accept a state mapping")
	}
	var err error
	switch {
	case kind == "RELAY":
		return apperr.New("A relay uses named status entries instead of a single state mapping")
	case kind == MeasurementKind:
		return apperr.New("A measurement list uses its own measurement rows instead of a single state mapping")
	case isSwitch(kind):
		switch mode {
		case "direct":
			_, err = validateSignalBinding(source, ieds, "Dbpos")
		case "single_to_dbpos":
			_, err = validateSignalBinding(source, ieds, "BOOLEAN")
		default:
			opened := object(pyval.Lookup(mapping, "open", nil))
			closed := object(pyval.Lookup(mapping, "closed", nil))
			if text(opened, "ied_id") == text(closed, "ied_id") && text(opened, "ref") == text(closed, "ref") {
				return apperr.New("Open and closed contacts must be different signals")
			}
			if _, err = validateSignalBinding(opened, ieds, "BOOLEAN"); err == nil {
				_, err = validateSignalBinding(closed, ieds, "BOOLEAN")
			}
		}
	case kind == "TRIP_RELAY":
		if mode != "direct" {
			return apperr.New("Trip relay state requires one Boolean signal")
		}
		_, err = validateSignalBinding(source, ieds, "BOOLEAN")
	case kind == "PUSH_BUTTON":
		pushMode := pyval.Lookup(data, "push_mode", nil)
		if !pyval.InTuple(pushMode, "latching_status", "self_resetting") || mode != "direct" {
			return apperr.New("Only a latching or self-resetting push button accepts one Boolean state signal")
		}
		_, err = validateSignalBinding(source, ieds, "BOOLEAN")
	case kind == "SOURCE":
		if mode != "direct" {
			return apperr.New("Analog source requires one numeric MX signal")
		}
		signal, failure := validateSignalBinding(source, ieds, "")
		if failure != nil {
			return failure
		}
		if text(signal, "fc") != "MX" || !pyval.InTuple(text(signal, "kind"), numericKind...) {
			return apperr.New("Analog source requires one numeric MX signal")
		}
	case mode != "direct":
		return apperr.New("Constructed double-point state is only valid for CB, DS or ES equipment")
	default:
		_, err = validateSignalBinding(source, ieds, "")
	}
	return err
}

func validateControl(data *pyjson.Object, kind string, ieds *Lookup) error {
	raw := pyval.Lookup(data, "control_binding", nil)
	if !pyval.Truthy(raw) {
		return nil
	}
	binding := object(raw)
	if name, found := drawingOnly[kind]; found {
		return apperr.New(name + " is a drawing point and does not accept a control binding")
	}
	if !isSwitch(kind) && kind != "TRIP_RELAY" && kind != "PUSH_BUTTON" {
		return apperr.New("This equipment type does not accept a control binding")
	}
	if kind == "PUSH_BUTTON" && pyval.Lookup(data, "push_mode", nil) != "spc_momentary" {
		return apperr.New("Only a momentary command push button accepts a command binding")
	}
	var control *pyjson.Object
	if ied := ieds.IED(text(binding, "ied_id")); ied != nil {
		control = ieds.Entry(pyval.Lookup(ied, "controls", nil), text(binding, "ref"))
	}
	if control == nil {
		return apperr.New("Equipment references an unknown control")
	}
	expected := "SPC"
	if isSwitch(kind) {
		expected = "DPC"
	}
	if pyval.Lookup(control, "cdc", nil) != expected {
		return apperr.New(kind + " equipment requires a " + expected + " control")
	}
	return nil
}
