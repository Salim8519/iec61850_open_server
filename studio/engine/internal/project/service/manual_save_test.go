package service

import (
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/testutil"
)

// onDisk is the project name on disk after this run ends: the service is closed (process
// exit), the journal folded (next start) and the autosave opened.
func onDisk(t *testing.T, s *Service, dir string) string {
	t.Helper()
	must(t, s.Close())
	if _, err := autosave.Fold(dir, nil); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(filepath.Join(dir, autosave.AutosaveName))
	if err != nil {
		t.Fatal(err)
	}
	reopened := newService(t)
	must(t, reopened.Restore(payload, false))
	return str(reopened.Project(), "name")
}

func serviceIn(t *testing.T, dir string) *Service {
	t.Helper()
	s, err := New(Config{DataDir: dir, Importer: &scl.Importer{Validator: loadSeed(t)}})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// rename is one change of the project.
func rename(t *testing.T, s *Service, name string) {
	t.Helper()
	_, err := s.Update(func(draft *pyjson.Object) (*pyjson.Object, error) { return draft.Set("name", name), nil })
	must(t, err)
}

// TestAutoSaveOffWritesOnlyOnSave (ADR-0139): with auto save off a change stays in the open
// project but not on disk until Save; leaving without Save gives back the last saved project.
func TestAutoSaveOffWritesOnlyOnSave(t *testing.T) {
	dir := testutil.TempDir(t)
	s := serviceIn(t, dir)
	rename(t, s, "saved")
	must(t, s.Save())
	must(t, s.SetAutoSave(false))
	rename(t, s, "not saved")
	if !s.Unsaved() || str(s.Project(), "name") != "not saved" {
		t.Fatalf("unsaved %v, name %q", s.Unsaved(), str(s.Project(), "name"))
	}
	if got := onDisk(t, s, dir); got != "saved" {
		t.Fatalf("leaving without Save kept %q, want the saved project", got)
	}
}

// TestSaveAndTurningAutoSaveOnWrite: Save writes the unsaved changes; turning auto save on
// writes them too, and later changes are journaled again so they survive a crash.
func TestSaveAndTurningAutoSaveOnWrite(t *testing.T) {
	dir := testutil.TempDir(t)
	s := serviceIn(t, dir)
	must(t, s.SetAutoSave(false))
	rename(t, s, "one")
	must(t, s.Save())
	if s.Unsaved() {
		t.Fatal("still unsaved after Save")
	}
	rename(t, s, "two")
	must(t, s.SetAutoSave(true))
	if s.Unsaved() || s.ManualSave() {
		t.Fatalf("unsaved %v manual %v after turning auto save on", s.Unsaved(), s.ManualSave())
	}
	rename(t, s, "three")
	if got := onDisk(t, s, dir); got != "three" {
		t.Fatalf("on disk %q, want three", got)
	}
}
