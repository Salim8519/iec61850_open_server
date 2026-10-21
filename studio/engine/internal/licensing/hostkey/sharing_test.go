package hostkey

import (
	"errors"
	"testing"
)

type fakeProcess struct {
	running bool
	started []string
	err     error
}

func (f *fakeProcess) Running() bool { return f.running }
func (f *fakeProcess) Start(executable string) error {
	f.started = append(f.started, executable)
	f.running = f.err == nil
	return f.err
}

func newSharing(found bool) (Sharing, *MemoryRunKey, *fakeProcess) {
	run, process := &MemoryRunKey{}, &fakeProcess{}
	return Sharing{Run: run, Process: process, Executable: `C:\Program Files\ScadaBench\scadabench_hostkey.exe`,
		Exists: func(string) bool { return found }}, run, process
}

func TestTurningSharingOnWritesTheRunValueAndStartsTheHelper(t *testing.T) {
	sharing, run, process := newSharing(true)
	if state := sharing.State(); state.Enabled || state.Running || !state.HelperFound {
		t.Fatalf("a fresh PC: %+v", state)
	}
	if err := sharing.Enable(); err != nil {
		t.Fatal(err)
	}
	if value, set := run.Get(); !set || value != `"C:\Program Files\ScadaBench\scadabench_hostkey.exe"` {
		t.Fatalf("the Run value is %q (%v): a path with spaces must be quoted", value, set)
	}
	if len(process.started) != 1 {
		t.Fatalf("the helper was started %d times", len(process.started))
	}
	if state := sharing.State(); !state.Enabled || !state.Running {
		t.Fatalf("state after turning on: %+v", state)
	}
	if err := sharing.Enable(); err != nil || len(process.started) != 1 {
		t.Fatalf("turning it on twice started the helper again: %v %v", err, process.started)
	}
}

func TestTurningSharingOffRemovesTheRunValueAndTheHelperSeesIt(t *testing.T) {
	sharing, run, _ := newSharing(true)
	sharing.Enable()
	if !sharing.Wanted() {
		t.Fatal("the helper would end while sharing is on")
	}
	if err := sharing.Disable(); err != nil {
		t.Fatal(err)
	}
	if _, set := run.Get(); set || sharing.Wanted() {
		t.Fatal("the Run value is still there: the helper keeps serving and starts at the next logon")
	}
	if err := sharing.Disable(); err != nil {
		t.Fatalf("turning off twice failed: %v", err)
	}
}

func TestACopyWithoutTheHelperCannotTurnSharingOn(t *testing.T) {
	sharing, run, process := newSharing(false)
	if err := sharing.Enable(); !errors.Is(err, ErrNoHelper) {
		t.Fatalf("err %v", err)
	}
	if _, set := run.Get(); set || len(process.started) != 0 {
		t.Fatal("something was changed although the helper is missing")
	}
	if sharing.State().HelperFound {
		t.Fatal("the missing helper was reported as found")
	}
}

func TestAHelperThatCannotStartLeavesTheValueForTheNextLogon(t *testing.T) {
	sharing, run, process := newSharing(true)
	process.err = errors.New("access denied")
	if err := sharing.Enable(); err == nil {
		t.Fatal("a start failure was hidden")
	}
	if _, set := run.Get(); !set {
		t.Fatal("the Run value was rolled back: nothing would start it at the next logon")
	}
}

func TestAnUpdatedOrMovedCopyFixesTheRunValueAtStartAndRestartsTheHelper(t *testing.T) {
	sharing, run, process := newSharing(true)
	run.Set(`"C:\Old\Place\scadabench_hostkey.exe"`)
	if err := sharing.Refresh(); err != nil {
		t.Fatal(err)
	}
	if value, _ := run.Get(); value != `"C:\Program Files\ScadaBench\scadabench_hostkey.exe"` || len(process.started) != 1 {
		t.Fatalf("value %q, started %v", value, process.started)
	}
	if err := sharing.Refresh(); err != nil || len(process.started) != 1 {
		t.Fatalf("a running helper was started again: %v", process.started)
	}
}

func TestStartingTheAppNeverTurnsSharingOn(t *testing.T) {
	sharing, run, process := newSharing(true)
	if err := sharing.Refresh(); err != nil {
		t.Fatal(err)
	}
	if _, set := run.Get(); set || len(process.started) != 0 {
		t.Fatal("Refresh turned sharing on by itself")
	}
}
