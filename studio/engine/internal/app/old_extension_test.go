package app

import (
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/project"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/store/projectfile"
	"studio-engine/internal/testutil"
)

// oldDataFolder is a data folder as an earlier ScadaBench left it: autosave.subsim holding a
// saved project called name, and a recovery copy, both with the old extension (ADR-0096).
func oldDataFolder(t *testing.T, name string) (dir string, saved []byte) {
	t.Helper()
	saved = projectBytes(t, name)
	dir = testutil.TempDir(t)
	put(t, filepath.Join(dir, "autosave.subsim"), saved)
	put(t, filepath.Join(dir, "recovery", "before-ied-delete-1.subsim"), []byte("backup"))
	return dir, saved
}

func projectBytes(t *testing.T, name string) []byte {
	t.Helper()
	payload, err := archive.NewEncoder().Encode(project.Compact(project.NewProject("0123456789abcdef0123456789abcdef").
		Set("name", name)))
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func put(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func projectName(t *testing.T, engine *Engine) any {
	t.Helper()
	return field(mustCall(t, engine, "GET", "/api/v1/project", nil), "name")
}

// TestOldExtensionFilesAreRenamedAtStartup: the autosave of an earlier version opens as the same
// project and nothing is lost.
func TestOldExtensionFilesAreRenamedAtStartup(t *testing.T) {
	dir, saved := oldDataFolder(t, "Legacy")
	engine := composeAt(t, dir, testutil.TempDir(t))
	defer engine.Close()
	project := mustCall(t, engine, "GET", "/api/v1/project", nil)
	if field(project, "startup_recovery", "status") != "restored" || field(project, "name") != "Legacy" {
		t.Fatalf("the old autosave did not open as the same project: %s", pyjsonText(project))
	}
	again, err := os.ReadFile(filepath.Join(dir, projectfile.AutosaveName))
	if err != nil || string(again) != string(saved) {
		t.Fatalf("autosave.scadabench must hold the old bytes: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "autosave.subsim")); !os.IsNotExist(err) {
		t.Fatal("autosave.subsim must be renamed, not copied")
	}
	if data, err := os.ReadFile(filepath.Join(dir, "recovery", "before-ied-delete-1.scadabench")); err != nil || string(data) != "backup" {
		t.Fatalf("recovery copy not renamed with its bytes: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "logs", projectfile.JournalName)); err != nil {
		t.Fatalf("the rename must be journaled: %v", err)
	}
}

// TestBothNamesPresentKeepsBothAndOpensTheNewOne: the .scadabench file is used, the old one is untouched.
func TestBothNamesPresentKeepsBothAndOpensTheNewOne(t *testing.T) {
	dir, old := oldDataFolder(t, "Older")
	put(t, filepath.Join(dir, projectfile.AutosaveName), projectBytes(t, "Newer"))
	put(t, filepath.Join(dir, "recovery", "before-ied-delete-1.scadabench"), []byte("already here"))
	engine := composeAt(t, dir, testutil.TempDir(t))
	defer engine.Close()
	if name := projectName(t, engine); name != "Newer" {
		t.Fatalf("the .scadabench file must win, got %v", name)
	}
	if kept, err := os.ReadFile(filepath.Join(dir, "autosave.subsim")); err != nil || string(kept) != string(old) {
		t.Fatalf("the old autosave must stay untouched: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "recovery", "before-ied-delete-1.subsim")); string(data) != "backup" {
		t.Fatal("the old recovery copy must stay untouched")
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "recovery", "before-ied-delete-1.scadabench")); string(data) != "already here" {
		t.Fatal("the new recovery copy must not be overwritten")
	}
}

// TestAnAutosaveThatCannotBeRenamedStopsStartup: on Windows a file held open by another program
// cannot be renamed; ScadaBench stops with a plain message and leaves the file where it is.
func TestAnAutosaveThatCannotBeRenamedStopsStartup(t *testing.T) {
	dir, saved := oldDataFolder(t, "Held")
	held, err := os.Open(filepath.Join(dir, "autosave.subsim"))
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if _, err := Compose(Options{DataDir: dir, NativeDir: testutil.TempDir(t), LicenseHome: testutil.TempDir(t),
		Token: "token", SecretKey: "secret", Version: "0.0.0+dev", NoBackground: true, Logf: t.Logf}); err == nil {
		t.Skip("this system lets a file that is open be renamed")
	} else if data, readErr := os.ReadFile(filepath.Join(dir, "autosave.subsim")); readErr != nil || string(data) != string(saved) {
		t.Fatalf("the autosave must stay where it was (%v) after: %v", readErr, err)
	}
}
