package control

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

// Every typed class in every control model: Select (when the model has one) then Operate,
// and the simulated feedback follows (ADR-0104).
func TestTypedClassesInEveryControlModel(t *testing.T) {
	r := startRig(t)
	names := map[int]string{1: "direct-normal", 2: "sbo-normal", 3: "direct-enhanced", 4: "sbo-enhanced"}
	type command struct {
		object, kind string
		value        any
		feedback     string
		want         float64
	}
	for model := 1; model <= 4; model++ {
		n := strconv.Itoa(model)
		commands := []command{
			{"GGIO" + n + ".Mod", "enum", int64(3), "GGIO" + n + ".Mod.stVal", 3},
			{"GGIO" + n + ".ISCSO1", "int", int64(7), "GGIO" + n + ".ISCSO1.stVal", 7},
			{"GGIO" + n + ".AnOut1", "float", 42.5, "GGIO" + n + ".AnOut1.mxVal.f", 42.5},
			{"ATCC" + n + ".TapChg", "step", "higher", "ATCC" + n + ".TapChg.valWTr.posVal", 6},
			{"ATCC" + n + ".TapChg", "step", "lower", "ATCC" + n + ".TapChg.valWTr.posVal", 5},
			{"ATCC" + n + ".TapSet", "int", int64(9), "ATCC" + n + ".TapSet.valWTr.posVal", 9},
			{"ATCC" + n + ".BndSet", "step", "higher", "ATCC" + n + ".BndSet.mxVal.f", 2.5},
		}
		for _, c := range commands {
			sbo := model == 2 || model == 4
			selected := r.step(t, "typed_select", c.object, c.kind, c.value)
			if field(selected, "ok") != true || field(selected, "model") != names[model] ||
				field(selected, "selected") != sbo {
				t.Fatalf("select %s: %v", c.object, selected)
			}
			if sbo && number(field(selected, "expires_ms")) != 3000 {
				t.Fatalf("select %s expires: %v", c.object, selected)
			}
			result := r.step(t, "typed_operate", c.object, c.kind, c.value, "require_select", sbo)
			if field(result, "ok") != true || field(result, "operated") != true {
				t.Fatalf("operate %s %v (%s): %v", c.object, c.value, names[model], result)
			}
			if (model == 3 || model == 4) && field(result, "terminated") != true {
				t.Fatalf("enhanced %s must end with a positive termination: %v", c.object, result)
			}
			if got := number(r.read(t, c.feedback)); got != c.want {
				t.Fatalf("%s after %v: %v, want %v", c.feedback, c.value, got, c.want)
			}
			applied := r.lastApplied(c.object)
			if field(applied, "or_ident") != "rig-test" || number(field(applied, "or_cat")) != 2 {
				t.Fatalf("originator must be logged: %v", applied)
			}
		}
		if r.read(t, "ATCC"+n+".TapChg.valWTr.transInd") != false {
			t.Fatal("the simulated tap changer is never left in transition")
		}
	}
}

func TestTypedLimitsAndValueChecks(t *testing.T) {
	r := startRig(t)
	refused := r.step(t, "typed_operate", "GGIO1.ISCSO1", "int", int64(11))
	if field(refused, "ok") != false || number(field(refused, "add_cause")) != 26 {
		t.Fatalf("INC above maxVal: %v", refused)
	}
	refused = r.step(t, "typed_operate", "GGIO3.Mod", "enum", int64(9))
	if field(refused, "ok") != false {
		t.Fatalf("ENC outside its enumeration: %v", refused)
	}
	for i := 0; i < 70; i++ {
		r.step(t, "typed_operate", "ATCC1.TapChg", "step", "higher")
	}
	if number(r.read(t, "ATCC1.TapChg.valWTr.posVal")) != 63 {
		t.Fatal("BSC stops at its upper limit 63")
	}
	limited := r.step(t, "typed_operate", "ATCC3.TapSet", "int", int64(18))
	if field(limited, "ok") != false {
		t.Fatalf("ISC above maxVal 17: %v", limited)
	}
	if _, err := r.client.Typed("typed_operate", r.fields("GGIO1.ISCSO1").Set("value", "x").Set("kind", "int")); err == nil ||
		!strings.Contains(err.Error(), "whole number") {
		t.Fatalf("a text value for INC: %v", err)
	}
	if _, err := r.client.Typed("typed_operate", r.fields("GGIO1.AnOut1").Set("value", true).Set("kind", "bool")); err == nil {
		t.Fatal("APC does not take on/off")
	}
}

func TestTypedCancelTimeoutAndTest(t *testing.T) {
	r := startRig(t)
	for _, object := range []string{"GGIO2.ISCSO1", "GGIO4.ISCSO1"} {
		r.step(t, "typed_select", object, "int", int64(4))
		canceled := r.step(t, "typed_cancel", object, "", nil)
		if field(canceled, "ok") != true || field(canceled, "canceled") != true {
			t.Fatalf("cancel %s: %v", object, canceled)
		}
		after := r.step(t, "typed_operate", object, "int", int64(4), "require_select", true)
		if field(after, "ok") != false || field(after, "error") != "Press Select first." {
			t.Fatalf("operate after cancel %s: %v", object, after)
		}
	}
	r.step(t, "typed_select", "GGIO2.AnOut1", "float", 10.0)
	time.Sleep(3300 * time.Millisecond)
	expired := r.step(t, "typed_operate", "GGIO2.AnOut1", "float", 10.0, "require_select", true)
	if field(expired, "error_code") != "select_expired" || !strings.Contains(field(expired, "error").(string), "Select expired after 3 s") {
		t.Fatalf("operate after the select timeout: %v", expired)
	}
	tested := r.step(t, "typed_operate", "GGIO1.ISCSO1", "int", int64(5), "test", true)
	if field(tested, "ok") != true || number(r.read(t, "GGIO1.ISCSO1.stVal")) != 0 {
		t.Fatalf("a test command is acknowledged and changes nothing: %v", tested)
	}
}

func TestTypedInterlockAndSynchroBlocks(t *testing.T) {
	r := startRig(t)
	r.checks(t, "GGIO1.ISCSO1", "interlock_blocked", true)
	blocked := r.step(t, "typed_operate", "GGIO1.ISCSO1", "int", int64(2), "interlock_check", true)
	if field(blocked, "ok") != false || number(field(blocked, "add_cause")) != 10 {
		t.Fatalf("interlock check against a simulated block: %v", blocked)
	}
	free := r.step(t, "typed_operate", "GGIO1.ISCSO1", "int", int64(2))
	if field(free, "ok") != true {
		t.Fatalf("without interlock check the command passes: %v", free)
	}
	r.checks(t, "GGIO3.AnOut1", "synchro_blocked", true)
	synchro := r.step(t, "typed_operate", "GGIO3.AnOut1", "float", 5.0, "synchro_check", true)
	if field(synchro, "ok") != false || number(field(synchro, "add_cause")) != 11 {
		t.Fatalf("synchro check against a simulated block: %v", synchro)
	}
	r.checks(t, "ATCC4.TapChg", "blocked", true)
	if field(r.step(t, "typed_operate", "ATCC4.TapChg", "step", "higher"), "ok") != false {
		t.Fatal("a blocked control refuses every command")
	}
}
