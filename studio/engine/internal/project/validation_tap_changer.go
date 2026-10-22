package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// TapTransformerKind is the power transformer with an on-load tap changer (ADR-0109). Python
// does not know this kind.
const TapTransformerKind = "TAP_TRANSFORMER"

// tapBindings are the signal and control fields of a tap changer, in contract order.
var (
	tapSignals  = []string{"tap_source", "mode_source", "voltage_source"}
	tapControls = []string{"step_control", "raise_control", "lower_control", "mode_control"}
)

// integerKinds are the signal kinds a tap position or an integer mode status may have.
var integerKinds = []string{"INT8", "INT16", "INT32", "INT64", "INT8U", "INT16U", "INT24U", "INT32U"}

// checkTapChanger rejects tap changer settings on other kinds and checks a transformer's tap
// range, bindings and regulator settings. Messages say what to change.
func checkTapChanger(data *pyjson.Object, kind string, ieds *Lookup) error {
	tap := object(pyval.Lookup(data, "tap_changer", nil))
	if kind != TapTransformerKind {
		if tap != nil {
			return apperr.New("Only a tap-changer transformer accepts tap changer settings")
		}
		return nil
	}
	if pyval.Truthy(pyval.Lookup(data, "state_mapping", nil)) {
		return apperr.New("A tap-changer transformer uses its own tap position binding instead of a single state mapping")
	}
	if tap == nil {
		return nil // a fresh node gets its defaults in settleTapChanger
	}
	if err := checkTapRange(tap); err != nil {
		return err
	}
	if err := checkTapSignals(tap, ieds); err != nil {
		return err
	}
	if err := checkTapCommands(tap, ieds); err != nil {
		return err
	}
	return checkAvr(tap)
}

func whole(object *pyjson.Object, key string) int64 {
	value, _ := number(pyval.Lookup(object, key, nil))
	return int64(value)
}

func checkTapRange(tap *pyjson.Object) error {
	low, high, nominal := whole(tap, "min_tap"), whole(tap, "max_tap"), whole(tap, "nominal_tap")
	if low >= high {
		return apperr.New("The lowest tap must be smaller than the highest tap, for example 1 and 17")
	}
	if nominal < low || nominal > high {
		return apperr.New("The nominal tap (the tap with the rated ratio) must be between the lowest and the highest tap")
	}
	return nil
}

// checkTapSignals checks the tap position (integer ST), the mode status (Boolean or integer ST)
// and the LV voltage (numeric MX).
func checkTapSignals(tap *pyjson.Object, ieds *Lookup) error {
	for _, key := range tapSignals {
		binding := object(pyval.Lookup(tap, key, nil))
		if binding == nil {
			continue
		}
		signal, err := bindingSignal(binding, ieds)
		if err != nil {
			return err
		}
		have, fc, ref := text(signal, "kind"), text(signal, "fc"), text(signal, "ref")
		switch key {
		case "tap_source":
			if !pyval.InTuple(have, integerKinds...) || fc != "ST" {
				return apperr.New("The tap position needs a whole-number status (functional constraint ST), such as " +
					"valWTr.posVal of a BSC or stVal of an INS or ISC, but " + ref + " is " + have + " (" + fc + ")")
			}
		case "mode_source":
			if (have != "BOOLEAN" && !pyval.InTuple(have, integerKinds...)) || fc != "ST" {
				return apperr.New("The Auto/Manual status needs an on/off or whole-number status (ST), such as stVal of an " +
					"SPS, SPC or ENS, but " + ref + " is " + have + " (" + fc + ")")
			}
		default:
			if !pyval.InTuple(have, numericKind...) || fc != "MX" {
				return apperr.New("The LV voltage needs a measured number (functional constraint MX), such as mag.f of an " +
					"MMXU phase voltage, but " + ref + " is " + have + " (" + fc + ")")
			}
		}
	}
	return nil
}

// tapControl finds a control by ref among the IED's controls and its typed (extended) controls.
func tapControl(binding *pyjson.Object, ieds *Lookup) *pyjson.Object {
	ied := ieds.IED(text(binding, "ied_id"))
	if ied == nil {
		return nil
	}
	ref := text(binding, "ref")
	if control := ieds.Entry(pyval.Lookup(ied, "controls", nil), ref); control != nil {
		return control
	}
	return findByRef(extendedControlsOf(ied), ref) // ADR-0104: read from the stored SCL
}

// checkTapCommands checks that the command mode and its controls match: one BSC/ISC step
// control, or two different SPC controls, or nothing.
func checkTapCommands(tap *pyjson.Object, ieds *Lookup) error {
	present := map[string]*pyjson.Object{}
	for _, key := range tapControls {
		if binding := object(pyval.Lookup(tap, key, nil)); binding != nil {
			control := tapControl(binding, ieds)
			if control == nil {
				return apperr.New("The tap changer uses a control this IED does not have: " + text(binding, "ref") +
					". Choose the control again or import the IED again")
			}
			present[key] = control
		}
	}
	mode := text(tap, "command_mode")
	step, raise, lower := present["step_control"], present["raise_control"], present["lower_control"]
	switch mode {
	case "step": // the control may still be missing while it is being chosen (tapCommandReady)
		if raise != nil || lower != nil {
			return apperr.New("A step command uses one BSC or ISC control and no raise/lower pair")
		}
		if class := text(step, "cdc"); step != nil && class != "BSC" && class != "ISC" {
			return apperr.New("A step command needs a BSC or ISC control, but " + text(step, "ref") + " is " + class)
		}
	case "spc_pair": // either command may still be missing while they are being chosen
		if step != nil {
			return apperr.New("A raise/lower pair uses two single commands (SPC) and no step control")
		}
		if raise != nil && lower != nil && text(raise, "ref") == text(lower, "ref") {
			return apperr.New("Raise and lower must be two different commands")
		}
		for _, control := range []*pyjson.Object{raise, lower} {
			if control == nil {
				continue
			}
			if class := text(control, "cdc"); class != "SPC" {
				return apperr.New("Raise and lower need single commands (SPC), but " + text(control, "ref") + " is " + class)
			}
		}
	default:
		if step != nil || raise != nil || lower != nil {
			return apperr.New("Choose how the tap changer is commanded before choosing its controls")
		}
	}
	if control := present["mode_control"]; control != nil {
		if class := text(control, "cdc"); class != "SPC" && class != "ENC" {
			return apperr.New("The Auto/Manual command needs an SPC or ENC control, but " + text(control, "ref") + " is " + class)
		}
	}
	return nil
}

// tapCommandReady says whether raise and lower can be sent: a step control, or both SPCs.
func tapCommandReady(tap *pyjson.Object) bool {
	set := func(key string) bool { return object(pyval.Lookup(tap, key, nil)) != nil }
	switch text(tap, "command_mode") {
	case "step":
		return set("step_control")
	case "spc_pair":
		return set("raise_control") && set("lower_control")
	}
	return false
}
