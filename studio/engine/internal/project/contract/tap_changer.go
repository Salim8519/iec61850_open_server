package contract

import "studio-engine/internal/pyjson"

// The tap-changer transformer (ADR-0109) is a Go-only diagram kind: a two-winding power
// transformer with an on-load tap changer (RTCC). Its settings live in one omittable object,
// `tap_changer`, so documents without such a transformer keep their exact bytes.

// tapTransformerKind is the kind name; the project and runtime packages repeat it.
const tapTransformerKind = "TAP_TRANSFORMER"

// TapLimit bounds the tap numbers (min, max and nominal tap) to -64..64; real tap changers have
// at most about 35 positions, and BSC valWTr.posVal is an INT8.
const TapLimit = 64

// avrSettings are the simulated voltage regulator (90) of the transformer. Voltages are in kV,
// the band and step in percent, the delay in seconds. Signs and ranges that need words (a target
// above zero, HV and LV ratings, ...) are checked by the project package.
var avrSettings = model("AvrSettings", true,
	defaulted("enabled", boolean, false),
	defaulted("target_kv", bounded(1000000), 11.0),
	defaulted("bandwidth_pct", bounded(50), 1.5),
	defaulted("delay_s", bounded(3600), 30.0),
	defaulted("step_pct", bounded(20), 1.25),
	defaulted("hv_rated_kv", bounded(1000000), 132.0),
	defaulted("lv_rated_kv", bounded(1000000), 11.0),
	defaulted("source_kv", bounded(1000000), 132.0))

// avrDefaults is a fresh default regulator (every field at its default).
func avrDefaults() any {
	value, _ := avrSettings(&collector{}, loc{}, pyjson.NewObject())
	return value
}

// TapChangerDefaults is a fresh unbound tap changer with every default; the project package
// gives it to a new transformer that has none.
func TapChangerDefaults() *pyjson.Object {
	value, _ := tapChanger(&collector{}, loc{}, pyjson.NewObject())
	return value.(*pyjson.Object)
}

// tapChanger binds the tap changer to IED points:
//   - tap_source: the tap position (BSC valWTr.posVal, an INS/INC or ISC stVal; an integer ST);
//   - command_mode "step": step_control is one BSC or ISC control (raise = higher, lower = lower);
//     "spc_pair": raise_control and lower_control are two SPC controls; "none": no command;
//   - mode_source / mode_control: optional Auto/Manual status (Boolean true = Auto, or an integer
//     equal to auto_value) and command (SPC, or ENC with auto_value / manual_value);
//   - voltage_source: optional LV voltage measurement (numeric MX);
//   - min_tap, max_tap, nominal_tap: the tap range and the tap with the rated ratio.
var tapChanger = model("TapChanger", true,
	defaulted("tap_source", optional(signalBinding), nil),
	defaulted("command_mode", literal("none", "step", "spc_pair"), "none"),
	defaulted("step_control", optional(controlBinding), nil),
	defaulted("raise_control", optional(controlBinding), nil),
	defaulted("lower_control", optional(controlBinding), nil),
	defaulted("mode_source", optional(signalBinding), nil),
	defaulted("mode_control", optional(controlBinding), nil),
	defaulted("auto_value", integer(true, -1000, 1000), int64(1)),
	defaulted("manual_value", integer(true, -1000, 1000), int64(0)),
	defaulted("voltage_source", optional(signalBinding), nil),
	defaulted("min_tap", integer(true, -TapLimit, TapLimit), int64(1)),
	defaulted("max_tap", integer(true, -TapLimit, TapLimit), int64(17)),
	defaulted("nominal_tap", integer(true, -TapLimit, TapLimit), int64(9)),
	field{name: "avr", check: avrSettings, fallback: avrDefaults})
