package projectfile

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"studio-engine/internal/testutil"
)

// On Windows a file another program holds open cannot be renamed. An autosave in that state
// stops startup with a plain message and stays where it is; a recovery copy is only journaled.
func TestAnAutosaveThatCannotBeRenamedStopsStartupAndStaysPut(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("holding a file open blocks a rename only on Windows")
	}
	dir := testutil.TempDir(t)
	put(t, filepath.Join(dir, "autosave.subsim"), "project")
	put(t, filepath.Join(dir, "recovery", "a.subsim"), "backup")
	held, err := os.Open(filepath.Join(dir, "autosave.subsim"))
	if err != nil {
		t.Fatal(err)
	}
	records, migrateErr := Migrate(dir)
	_ = held.Close()
	if migrateErr == nil || !strings.Contains(migrateErr.Error(), "Nothing was deleted") ||
		len(records) != 1 || records[0].Kind != KindRenameFailed || records[0].Error == "" {
		t.Fatalf("records %v, error %v", records, migrateErr)
	}
	if read(t, filepath.Join(dir, "autosave.subsim")) != "project" {
		t.Fatal("the autosave must stay where it was")
	}
	if _, err := os.Stat(filepath.Join(dir, "recovery", "a.scadabench")); !os.IsNotExist(err) {
		t.Fatal("startup stops at the autosave; later files wait for the next start")
	}
}

func TestARecoveryCopyThatCannotBeRenamedDoesNotStopStartup(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("holding a file open blocks a rename only on Windows")
	}
	dir := testutil.TempDir(t)
	put(t, filepath.Join(dir, "recovery", "a.subsim"), "backup")
	held, err := os.Open(filepath.Join(dir, "recovery", "a.subsim"))
	if err != nil {
		t.Fatal(err)
	}
	records, migrateErr := Migrate(dir)
	_ = held.Close()
	if migrateErr != nil || len(records) != 1 || records[0].Kind != KindRenameFailed {
		t.Fatalf("records %v, error %v", records, migrateErr)
	}
	if read(t, filepath.Join(dir, "recovery", "a.subsim")) != "backup" {
		t.Fatal("the copy must stay where it was")
	}
}
