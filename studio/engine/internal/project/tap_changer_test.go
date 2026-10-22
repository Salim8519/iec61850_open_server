package project

import (
	"strings"
	"testing"

	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Tap-changer transformer (ADR-0109): contract defaults, cross-checks and IED removal.

const (
	tapRef     = "LD0/YLTC1.TapChg.valWTr.posVal"
	modeRef    = "LD0/ATCC1.Auto.stVal"
	voltRef    = "LD0/MMXU1.PhV.phsA.cVal.mag.f"
	raiseRef   = "LD0/GGIO1.SPCSO1"
	lowerRef   = "LD0/GGIO1.SPCSO2"
	bscRef     = "LD0/YLTC1.TapChg"
	breakerRef = "LD0/CSWI1.Pos"
)

func tapIEDs() []*pyjson.Object {
	signal := func(ref, kind, fc string) any { return o("ref", ref, "kind", kind, "fc", fc, "writable", true) }
	control := func(ref, cdc string) any { return o("ref", ref, "cdc", cdc, "status", ref+".stVal") }
	return []*pyjson.Object{o("id", "rtcc", "name", "RTCC1",
		"signals", []any{signal(tapRef, "INT8", "ST"), signal(modeRef, "BOOLEAN", "ST"), signal(voltRef, "FLOAT32", "MX"),
			signal("LD0/YLTC1.TapChg.valWTr.transInd", "BOOLEAN", "ST"), signal("LD0/LLN0.Cfg.setVal", "INT32", "CF")},
		"controls", []any{control(raiseRef, "SPC"), control(lowerRef, "SPC"), control(breakerRef, "DPC")},
		"extended_controls", []any{control(bscRef, "BSC"), control("LD0/ATCC1.Mod", "ENC")})}
}

// The tap fixtures carry their typed controls in the IED itself; in the app they come only from
// the stored SCL (ADR-0104, extendedControlsOf). The package's tests read that fixture key first.
func init() {
	fromSCL := extendedControlsOf
	extendedControlsOf = func(ied *pyjson.Object) any {
		if list, present := ied.Get("extended_controls"); present {
			return list
		}
		return fromSCL(ied)
	}
}

func binding(ref string) *pyjson.Object { return o("ied_id", "rtcc", "ref", ref, "report_ref", nil) }
func ctl(ref string) *pyjson.Object     { return o("ied_id", "rtcc", "ref", ref) }

func validateTap(tap *pyjson.Object) (*pyjson.Object, error) {
	data := o()
	if tap != nil {
		data.Set("tap_changer", tap)
	}
	return ValidateDiagram(listDiagram(TapTransformerKind, data), tapIEDs())
}

func TestTapChangerDefaults(t *testing.T) {
	result, err := validateTap(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"tap_source": null, "command_mode": "none", "step_control": null, "raise_control": null, ` +
		`"lower_control": null, "mode_source": null, "mode_control": null, "auto_value": 1, "manual_value": 0, ` +
		`"voltage_source": null, "min_tap": 1, "max_tap": 17, "nominal_tap": 9, "avr": {"enabled": false, ` +
		`"target_kv": 11.0, "bandwidth_pct": 1.5, "delay_s": 30.0, "step_pct": 1.25, "hv_rated_kv": 132.0, ` +
		`"lv_rated_kv": 11.0, "source_kv": 132.0}}`
	if got := dumps(t, pyval.Lookup(firstData(t, result), "tap_changer", nil)); got != want {
		t.Fatalf("defaults\n got: %s\nwant: %s", got, want)
	}
	breaker, err := ValidateDiagram(listDiagram("CB", o()), tapIEDs())
	if err != nil {
		t.Fatal(err)
	}
	if _, present := firstData(t, breaker).Get("tap_changer"); present {
		t.Fatal("other kinds must not carry tap_changer (their saved bytes stay unchanged)")
	}
	if _, err := ValidateDiagram(listDiagram("CB", o("tap_changer", o())), tapIEDs()); studioMessage(err) !=
		"Only a tap-changer transformer accepts tap changer settings" {
		t.Fatalf("got %v", err)
	}
}

func TestTapChangerValidBindings(t *testing.T) {
	pair := o("tap_source", binding(tapRef), "command_mode", "spc_pair", "raise_control", ctl(raiseRef),
		"lower_control", ctl(lowerRef), "mode_source", binding(modeRef), "mode_control", ctl(raiseRef),
		"voltage_source", binding(voltRef), "avr", o("enabled", true, "delay_s", int64(10)))
	if _, err := validateTap(pair); err != nil {
		t.Fatalf("SPC pair: %v", err)
	}
	step := o("tap_source", binding(tapRef), "command_mode", "step", "step_control", ctl(bscRef),
		"mode_control", ctl("LD0/ATCC1.Mod"))
	if _, err := validateTap(step); err != nil {
		t.Fatalf("BSC step with ENC mode: %v", err)
	}
	half := o("command_mode", "spc_pair", "raise_control", ctl(raiseRef))
	if _, err := validateTap(half); err != nil {
		t.Fatalf("a pair still being chosen must save: %v", err)
	}
}

func TestTapChangerRefusals(t *testing.T) {
	cases := []struct {
		name string
		tap  *pyjson.Object
		want string
	}{
		{"range", o("min_tap", int64(5), "max_tap", int64(5)), "lowest tap must be smaller"},
		{"nominal", o("nominal_tap", int64(30)), "nominal tap"},
		{"tap kind", o("tap_source", binding(voltRef)), "whole-number status"},
		{"tap fc", o("tap_source", binding("LD0/LLN0.Cfg.setVal")), "whole-number status"},
		{"mode kind", o("mode_source", binding(voltRef)), "Auto/Manual status"},
		{"voltage kind", o("voltage_source", binding(tapRef)), "measured number"},
		{"unknown signal", o("tap_source", binding("LD0/X.posVal")), "unknown signal"},
		{"step needs BSC", o("command_mode", "step", "step_control", ctl(raiseRef)), "BSC or ISC control"},
		{"step with pair", o("command_mode", "step", "raise_control", ctl(raiseRef)), "one BSC or ISC control"},
		{"pair with step", o("command_mode", "spc_pair", "step_control", ctl(bscRef)), "no step control"},
		{"avr needs both", o("tap_source", binding(tapRef), "command_mode", "spc_pair", "raise_control", ctl(raiseRef),
			"avr", o("enabled", true)), "needs the raise and lower"},
		{"pair same", o("command_mode", "spc_pair", "raise_control", ctl(raiseRef), "lower_control", ctl(raiseRef)),
			"two different commands"},
		{"pair needs SPC", o("command_mode", "spc_pair", "raise_control", ctl(raiseRef), "lower_control", ctl(breakerRef)),
			"single commands (SPC)"},
		{"controls without mode", o("raise_control", ctl(raiseRef)), "Choose how the tap changer is commanded"},
		{"unknown control", o("command_mode", "step", "step_control", ctl("LD0/NONE")), "does not have"},
		{"mode control", o("mode_control", ctl(breakerRef)), "SPC or ENC"},
		{"avr target", o("avr", o("target_kv", 0)), "target voltage"},
		{"avr band", o("avr", o("bandwidth_pct", 0)), "band must be above 0"},
		{"avr delay", o("avr", o("delay_s", -1)), "delay cannot be negative"},
		{"avr ratings", o("avr", o("lv_rated_kv", 0)), "rated HV and LV"},
		{"avr needs tap", o("avr", o("enabled", true)), "needs the tap position"},
		{"avr needs command", o("tap_source", binding(tapRef), "avr", o("enabled", true)), "needs the raise and lower"},
	}
	for _, item := range cases {
		_, err := validateTap(item.tap)
		if !strings.Contains(studioMessage(err), item.want) {
			t.Errorf("%s: got %q, want it to contain %q", item.name, studioMessage(err), item.want)
		}
	}
	mapped := listDiagram(TapTransformerKind, o("state_mapping", direct("rtcc", 0)))
	if _, err := ValidateDiagram(mapped, tapIEDs()); !strings.Contains(studioMessage(err), "own tap position") {
		t.Fatalf("state mapping: %v", err)
	}
	extra := o("tap_changer", o("bogus", true))
	if _, err := ValidateDiagram(listDiagram(TapTransformerKind, extra), tapIEDs()); err == nil {
		t.Fatal("unknown tap changer keys must be refused")
	}
}

func TestTapChangerIEDRemoval(t *testing.T) {
	result, err := validateTap(o("tap_source", binding(tapRef), "command_mode", "spc_pair",
		"raise_control", ctl(raiseRef), "lower_control", ctl(lowerRef), "avr", o("enabled", true)))
	if err != nil {
		t.Fatal(err)
	}
	data := firstData(t, result)
	if !usesIED(data, map[string]bool{"rtcc": true}) {
		t.Fatal("a bound transformer must count as using its IED")
	}
	dropTapChangerOf(data, map[string]bool{"rtcc": true})
	tap := object(pyval.Lookup(data, "tap_changer", nil))
	if text(tap, "command_mode") != "none" || pyval.Lookup(tap, "tap_source", nil) != nil ||
		pyval.Lookup(object(pyval.Lookup(tap, "avr", nil)), "enabled", nil) != false {
		t.Fatalf("after removal: %s", dumps(t, tap))
	}
	if _, err := ValidateDiagram(o("nodes", []any{o("id", "m1", "type", "equipment", "position", o("x", 0, "y", 0),
		"data", data)}, "edges", []any{}), nil); err != nil {
		t.Fatalf("the transformer must stay valid without the IED: %v", err)
	}
}
