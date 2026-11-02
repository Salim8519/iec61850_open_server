package service

import (
	"testing"
	"time"

	"studio-engine/internal/project/history"
	"studio-engine/internal/pyjson"
)

func panelOf(s *Service) int { return len(s.TestPanel()) }

func entry(name string) any { return pyjson.NewObject().Set("name", name) }

// spaced makes every recorded change its own step (no joining inside history.Window).
func spaced(s *Service) {
	clock := time.Unix(1_000_000, 0)
	s.history = history.New(undoRules{}, func() time.Time { clock = clock.Add(2 * history.Window); return clock })
}

// TestUndoRedoAcrossAreas: an IED import and a Test page change are two steps; Undo puts
// them back newest first, Redo applies them again, each as a normal announced change.
func TestUndoRedoAcrossAreas(t *testing.T) {
	s := newService(t)
	spaced(s)
	if _, err := s.AddTemplate("BayController"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveTestPanel([]any{entry("a")}); err != nil {
		t.Fatal(err)
	}
	status := s.HistoryStatus()
	if can, _ := status.Get("can_undo"); can != true {
		t.Fatal("cannot undo")
	}
	if label, _ := status.Get("undo"); label != "Test page change" {
		t.Fatalf("label %v", label)
	}
	revision := s.Revision()
	must(t, s.Undo())
	if panelOf(s) != 0 || len(s.IEDs()) != 1 || s.Revision() != revision+1 {
		t.Fatalf("after undo: panel %d, ieds %d", panelOf(s), len(s.IEDs()))
	}
	must(t, s.Undo())
	if len(s.IEDs()) != 0 {
		t.Fatalf("IED still there: %d", len(s.IEDs()))
	}
	if err := s.Undo(); err == nil {
		t.Fatal("undo past the first step")
	}
	must(t, s.Redo())
	must(t, s.Redo())
	if panelOf(s) != 1 || len(s.IEDs()) != 1 {
		t.Fatalf("after redo: panel %d, ieds %d", panelOf(s), len(s.IEDs()))
	}
	// A new change forgets what could be redone.
	must(t, s.Undo())
	must(t, s.SaveTestPanel([]any{entry("b"), entry("c")}))
	if can, _ := s.HistoryStatus().Get("can_redo"); can != false {
		t.Fatal("redo survived a new change")
	}
}

// TestUndoKeepsAutomaticParts: the gateway's own on/off memory is no step and survives Undo,
// and an IED's lab address placed later stays when an older IED step is undone.
func TestUndoKeepsAutomaticParts(t *testing.T) {
	s := newService(t)
	spaced(s)
	must(t, s.SaveGateway(s.Gateway().Set("auto_start", false)))
	if s.history.Keys(true) != nil {
		t.Fatal("auto_start alone became a step")
	}
	must(t, s.SaveTestPanel([]any{entry("a")}))
	if _, err := s.AddTemplate("BayController"); err != nil {
		t.Fatal(err)
	}
	must(t, s.SaveTestPanel([]any{entry("a"), entry("b")}))
	id, _ := s.IEDs()[0].Get("id")
	_, err := s.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
		list, _ := draft.Get("ieds")
		list.([]any)[0].(*pyjson.Object).Set("lab_ip", "192.168.50.7")
		return nil, nil
	})
	must(t, err)
	if keys := s.history.Keys(true); len(keys) != 1 || keys[0] != "test_panel" {
		t.Fatalf("a lab address became a step: %v", keys)
	}
	must(t, s.Undo()) // the second Test page change
	must(t, s.SaveGateway(s.Gateway().Set("auto_start", true)))
	must(t, s.Redo())
	if panelOf(s) != 2 {
		t.Fatalf("redo after an automatic gateway change: panel %d", panelOf(s))
	}
	if on, _ := s.Gateway().Get("auto_start"); on != true {
		t.Fatal("auto_start lost")
	}
	if ip, _ := s.IEDs()[0].Get("lab_ip"); ip != "192.168.50.7" {
		t.Fatalf("lab address lost: %v", ip)
	}
	if got, _ := s.IEDs()[0].Get("id"); got != id {
		t.Fatal("IED changed")
	}
}

// TestUndoWhileRunning: changes made while IEDs run are no steps, Undo is refused and the
// status says so.
func TestUndoWhileRunning(t *testing.T) {
	s := newService(t)
	spaced(s)
	must(t, s.SaveTestPanel([]any{entry("a")}))
	s.SetRunning(true)
	must(t, s.SaveTestPanel([]any{entry("a"), entry("b")}))
	if err := s.Undo(); err == nil {
		t.Fatal("undo while running")
	}
	if can, _ := s.HistoryStatus().Get("can_undo"); can != false {
		t.Fatal("status offers undo while running")
	}
	s.SetRunning(false)
	// The last step's result was changed while running: it can no longer apply.
	if can, _ := s.HistoryStatus().Get("can_undo"); can != false {
		t.Fatal("a stale step is offered")
	}
}

// TestQuickEditsJoinAndLimit: edits of the same thing within the window are one step; at most
// history.Limit steps are kept.
func TestQuickEditsJoinAndLimit(t *testing.T) {
	s := newService(t)
	for index := 0; index < 3; index++ {
		must(t, s.SaveTestPanel([]any{entry(string(rune('a' + index)))}))
	}
	must(t, s.Undo())
	if panelOf(s) != 0 {
		t.Fatalf("quick edits were not one step: panel %d", panelOf(s))
	}
	spaced(s)
	for index := 0; index < history.Limit+5; index++ {
		must(t, s.SaveTestPanel(make([]any, index+1)))
	}
	undone := 0
	for s.Undo() == nil {
		undone++
	}
	if undone != history.Limit || panelOf(s) != 5 {
		t.Fatalf("undid %d steps, panel %d", undone, panelOf(s))
	}
}

// TestNewProjectAndOpenStartAgain: a new project is empty, keeps this PC's network choice
// and has no steps; opening a file forgets the steps too.
func TestNewProjectAndOpenStartAgain(t *testing.T) {
	s := newService(t)
	spaced(s)
	if _, err := s.AddTemplate("BayController"); err != nil {
		t.Fatal(err)
	}
	network, _ := s.Project().Get("network")
	oldID, _ := s.Project().Get("id")
	payload, err := s.Export()
	must(t, err)
	must(t, s.NewProject("North yard"))
	if name, _ := s.Project().Get("name"); name != "North yard" || len(s.IEDs()) != 0 {
		t.Fatalf("new project %v with %d IEDs", name, len(s.IEDs()))
	}
	if kept, _ := s.Project().Get("network"); kept != network {
		t.Fatal("network choice not kept")
	}
	if id, _ := s.Project().Get("id"); id == oldID || s.Undo() == nil {
		t.Fatal("same id or undo into the closed project")
	}
	must(t, s.SaveTestPanel([]any{entry("a")}))
	must(t, s.Restore(payload, true))
	if s.Undo() == nil || len(s.IEDs()) != 1 {
		t.Fatal("undo across an opened file")
	}
	s.SetRunning(true)
	if s.NewProject("") == nil {
		t.Fatal("new project while running")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
