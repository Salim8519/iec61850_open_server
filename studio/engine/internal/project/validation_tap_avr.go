package project

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/contract"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// checkAvr checks the simulated voltage regulator (ADR-0109): positive voltages, band and step,
// a delay that is not negative, and when it is switched on, a tap position and a command.
func checkAvr(tap *pyjson.Object) error {
	avr := object(pyval.Lookup(tap, "avr", nil))
	if avr == nil {
		return nil
	}
	read := func(key string) float64 {
		value, _ := number(pyval.Lookup(avr, key, nil))
		return value
	}
	switch {
	case read("target_kv") <= 0:
		return apperr.New("The target voltage must be above 0 kV, for example 11")
	case read("hv_rated_kv") <= 0 || read("lv_rated_kv") <= 0:
		return apperr.New("The rated HV and LV voltages must be above 0 kV, for example 132 and 11")
	case read("source_kv") < 0:
		return apperr.New("The source (HV) voltage cannot be negative")
	case read("bandwidth_pct") <= 0:
		return apperr.New("The band must be above 0 %, for example 1.5 (the regulator waits while the voltage is " +
			"within 1.5 % of the target)")
	case read("step_pct") <= 0:
		return apperr.New("The voltage step per tap must be above 0 %, for example 1.25")
	case read("delay_s") < 0:
		return apperr.New("The time delay cannot be negative, for example 30 seconds")
	}
	if pyval.Lookup(avr, "enabled", nil) != true {
		return nil
	}
	if object(pyval.Lookup(tap, "tap_source", nil)) == nil {
		return apperr.New("The voltage regulator needs the tap position. Choose where the tap position comes from first")
	}
	if !tapCommandReady(tap) {
		return apperr.New("The voltage regulator needs the raise and lower command. Choose how the tap changer is " +
			"commanded first")
	}
	return nil
}

// settleTapChanger gives a fresh transformer its default (unbound) tap changer and drops the
// field from other kinds, so their saved bytes never change.
func settleTapChanger(data *pyjson.Object, kind string) {
	if kind != TapTransformerKind {
		data.Delete("tap_changer")
		return
	}
	if object(pyval.Lookup(data, "tap_changer", nil)) == nil {
		data.Set("tap_changer", contract.TapChangerDefaults())
	}
}

// tapChangerUsesIED says whether one of the transformer's bindings names one of the IEDs.
func tapChangerUsesIED(data *pyjson.Object, ids map[string]bool) bool {
	tap := object(pyval.Lookup(data, "tap_changer", nil))
	if tap == nil {
		return false
	}
	for _, key := range append(append([]string{}, tapSignals...), tapControls...) {
		if binding := object(pyval.Lookup(tap, key, nil)); binding != nil && ids[text(binding, "ied_id")] {
			return true
		}
	}
	return false
}

// dropTapChangerOf clears the bindings of removed IEDs. A command left without its controls
// becomes "none" and a regulator without a tap position or command is switched off, so the
// transformer stays valid; the transformer itself stays on the diagram.
func dropTapChangerOf(data *pyjson.Object, ids map[string]bool) {
	if data == nil || !tapChangerUsesIED(data, ids) {
		return
	}
	tap := object(pyval.Lookup(data, "tap_changer", nil))
	for _, key := range append(append([]string{}, tapSignals...), tapControls...) {
		if binding := object(pyval.Lookup(tap, key, nil)); binding != nil && ids[text(binding, "ied_id")] {
			tap.Set(key, nil)
		}
	}
	mode := text(tap, "command_mode")
	missing := (mode == "step" && object(pyval.Lookup(tap, "step_control", nil)) == nil) ||
		(mode == "spc_pair" && (object(pyval.Lookup(tap, "raise_control", nil)) == nil ||
			object(pyval.Lookup(tap, "lower_control", nil)) == nil))
	if missing {
		tap.Set("command_mode", "none")
		tap.Set("step_control", nil)
		tap.Set("raise_control", nil)
		tap.Set("lower_control", nil)
	}
	avr := object(pyval.Lookup(tap, "avr", nil))
	if avr != nil && (!tapCommandReady(tap) || object(pyval.Lookup(tap, "tap_source", nil)) == nil) {
		avr.Set("enabled", false)
	}
}
