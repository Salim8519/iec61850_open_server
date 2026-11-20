package service

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/testutil"
)

// A Save whose write fails says so in words the user can act on, as a runtime failure (503),
// never Flask's generic "internal error ... the server is overloaded" (fix plan A5, found live
// on Windows with a read-only and a locked autosave.scadabench).
func TestFailedSaveIsExplained(t *testing.T) {
	dir := testutil.TempDir(t)
	s := serviceIn(t, dir)
	t.Cleanup(func() { _ = s.Close() })
	rename(t, s, "first")
	must(t, s.Save())
	target := filepath.Join(dir, autosave.AutosaveName)
	must(t, os.Remove(target))
	must(t, os.MkdirAll(filepath.Join(target, "in the way"), 0o755)) // nothing can be renamed onto it
	must(t, s.SetAutoSave(false))                                    // Save is then the only write
	rename(t, s, "second")
	err := s.Save()
	var coded *apperr.Error
	if !errors.As(err, &coded) || coded.Kind != apperr.Runtime {
		t.Fatalf("Save answered %T %v, want a runtime failure in words", err, err)
	}
	if !strings.HasPrefix(coded.Message, "ScadaBench could not write the project file (") ||
		!strings.Contains(coded.Message, "press Save again") || strings.Contains(coded.Message, dir) {
		t.Fatalf("message %q", coded.Message)
	}
	if !s.Unsaved() {
		t.Fatal("a failed Save cleared the unsaved mark")
	}
	must(t, os.RemoveAll(target))
	must(t, s.Save())
}
