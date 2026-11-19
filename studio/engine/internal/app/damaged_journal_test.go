package app

import (
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"testing"

	"studio-engine/internal/store/projectfile"
	"studio-engine/internal/testutil"
)

// A4: a damaged autosave journal no longer stops start-up. The app composes, the journal is
// moved into the recovery folder (with a copy of the autosave), the autosave stays in place
// and GET /project carries one startup notice naming the moved journal.
func TestDamagedJournalStartsAndMovesItToRecovery(t *testing.T) {
	dir := testutil.TempDir(t)
	saved := projectBytes(t, "Kept")
	journal := []byte("{oops\n{oops\n")
	put(t, filepath.Join(dir, projectfile.AutosaveName), saved)
	put(t, filepath.Join(dir, "autosave.journal"), journal)
	engine := composeAt(t, dir, testutil.TempDir(t))
	defer engine.Close()
	project := mustCall(t, engine, "GET", "/api/v1/project", nil)
	if field(project, "startup_recovery", "status") != "unreadable" {
		t.Fatalf("expected the journal notice: %s", pyjsonText(project))
	}
	backup, _ := field(project, "startup_recovery", "backup").(string)
	if filepath.Dir(backup) != filepath.Join(dir, "recovery") || !strings.HasSuffix(backup, ".journal") {
		t.Fatalf("backup %q", backup)
	}
	if data, err := os.ReadFile(backup); err != nil || string(data) != string(journal) {
		t.Fatalf("journal not kept in recovery: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "autosave.journal")); !os.IsNotExist(err) {
		t.Fatal("the damaged journal must leave the data folder")
	}
	copied := strings.TrimSuffix(backup, ".journal") + projectfile.Extension
	if data, err := os.ReadFile(copied); err != nil || string(data) != string(saved) {
		t.Fatalf("autosave copy missing in recovery: %v", err)
	}
	if goruntime.GOOS == "windows" && field(project, "name") != "Kept" {
		t.Fatalf("the last autosave must open: %s", pyjsonText(project))
	}
}
