package acquisition

import (
	"errors"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// deviceWorker answers studio_live reads from a table; a ref it does not know answers an error,
// as a device does for an object it does not have.
type deviceWorker struct {
	values map[string]any
	asked  []string
	fail   error
}

func (w *deviceWorker) Request(op string, _ time.Duration, args *pyjson.Object) (any, error) {
	if w.fail != nil {
		return nil, w.fail
	}
	answers := []any{}
	for _, item := range pyv.Objects(pyv.GetList(args, "refs")) {
		ref := pyv.GetStr(item, "ref")
		w.asked = append(w.asked, ref+"@"+pyv.GetStr(item, "fc"))
		if value, found := w.values[ref]; found {
			answers = append(answers, pyv.O("ref", ref, "value", value))
		} else {
			answers = append(answers, pyv.O("ref", ref, "error", int64(10)))
		}
	}
	return pyv.O("values", answers), nil
}

func (w *deviceWorker) Close() {}

func checkIED() *pyjson.Object {
	control := func(ref, cdc string, model, timeout int64) any {
		return pyv.O("ref", ref, "cdc", cdc, "model", model, "sbo_timeout_ms", timeout)
	}
	signal := func(ref string) any { return pyv.O("ref", ref, "fc", "ST") }
	return pyv.O("id", "r1", "name", "R1", "controls", []any{
		control("LD/CSWI1.Pos", "DPC", 4, 30000), control("LD/CSWI2.Pos", "DPC", 1, 0),
		control("LD/CSWI3.Pos", "DPC", 2, 10000), control("LD/CSWI4.Pos", "DPC", 1, 0),
		control("LD/GGIO1.SPCSO1", "SPC", 1, 0)},
		"signals", []any{signal("LD/LLN0.Loc.stVal"), signal("LD/CSWI1.Loc.stVal"), signal("LD/CSWI9.Loc.stVal")})
}

// withTyped gives the IEDs typed controls without stored SCL.
func withTyped(t *testing.T, list ...any) {
	original := typedControlsOf
	typedControlsOf = func(*pyjson.Object) []any { return list }
	t.Cleanup(func() { typedControlsOf = original })
}

// Every CID control is classed against the device's own settings; the device wins and an
// unreadable value is reported, never filled from the CID (ADR-0138).
func TestControlCheckClassesEveryControl(t *testing.T) {
	withTyped(t, pyv.O("ref", "LD/ATCC1.TapChg", "cdc", "BSC", "model", int64(1), "sbo_timeout_ms", int64(0)),
		pyv.O("ref", "LD/CSWI1.Pos", "cdc", "DPC", "model", int64(4))) // listed twice: checked once
	device := &deviceWorker{values: map[string]any{
		"LD/CSWI1.Pos.ctlModel": int64(4), "LD/CSWI1.Pos.sboTimeout": int64(30000),
		"LD/CSWI2.Pos.ctlModel": int64(4),
		"LD/CSWI3.Pos.ctlModel": int64(2), "LD/CSWI3.Pos.sboTimeout": int64(5000),
		"LD/GGIO1.SPCSO1.ctlModel": int64(0), "LD/ATCC1.TapChg.ctlModel": int64(1),
		"LD/LLN0.Loc.stVal": true, "LD/CSWI1.Loc.stVal": false}}
	check, err := readControlCheck(device, checkIED(), time.Now)
	if err != nil {
		t.Fatal(err)
	}
	results := map[string]string{}
	for _, row := range pyv.Objects(pyv.GetList(check, "rows")) {
		results[pyv.GetStr(row, "ref")] = pyv.GetStr(row, "result") + ": " + pyv.GetStr(row, "note")
	}
	want := map[string]string{
		"LD/CSWI1.Pos":    "match: ",
		"LD/CSWI2.Pos":    "differs: The CID says direct, normal security; the device uses select before operate, enhanced security. ScadaBench sends with the device's own model.",
		"LD/CSWI3.Pos":    "differs: The selection lasts 5000 ms on the device, 10000 ms in the CID.",
		"LD/CSWI4.Pos":    "unreadable: the device could not read its control model (error 10)",
		"LD/GGIO1.SPCSO1": "differs: The device has this object as status only, so it refuses every command to it.",
		"LD/ATCC1.TapChg": "match: ",
	}
	if len(results) != len(want) {
		t.Fatalf("checked %d controls, want %d: %v", len(results), len(want), results)
	}
	for ref, text := range want {
		if results[ref] != text {
			t.Errorf("%s: %q, want %q", ref, results[ref], text)
		}
	}
	summary := pyv.Repr(pyv.GetObj(check, "summary"))
	if summary != "{'total': 6, 'match': 2, 'differs': 3, 'unreadable': 1, 'local': 1, 'loc_checked': 2}" {
		t.Fatalf("summary %s", summary)
	}
	if local := pyv.Repr(pyv.Get(check, "local")); local != "['LD/LLN0.Loc.stVal']" {
		t.Fatalf("Local is read from LLN0 and the control's own node only: %s", local)
	}
	asked := strings.Join(device.asked, " ")
	if strings.Contains(asked, "CSWI2.Pos.sboTimeout") || !strings.Contains(asked, "CSWI1.Pos.sboTimeout@CF") {
		t.Fatalf("sboTimeout is read only for SBO controls: %s", asked)
	}
	if strings.Contains(asked, "CSWI9") {
		t.Fatalf("a Loc of a node without a control is not read: %s", asked)
	}
}

// A device that cannot answer leaves the check's error, and the session keeps going.
func TestControlCheckFailureNeverEndsTheSession(t *testing.T) {
	withTyped(t)
	session := &RealSession{deps: Deps{Clock: testClock{start: time.Now()}}, ied: checkIED()}
	if session.Controls() != nil {
		t.Fatal("no check before the first read")
	}
	session.checkControls(&deviceWorker{fail: errors.New("Connection to the device was lost")})
	message := pyv.GetStr(session.Controls(), "error")
	if message != "ScadaBench could not read the device's control settings: Connection to the device was lost" {
		t.Fatalf("error %q", message)
	}
	session.checkControls(&deviceWorker{values: map[string]any{}})
	if pyv.Has(session.Controls(), "error") || len(pyv.GetList(session.Controls(), "rows")) != 5 {
		t.Fatalf("a later read replaces the error: %s", pyv.Repr(session.Controls()))
	}
}
