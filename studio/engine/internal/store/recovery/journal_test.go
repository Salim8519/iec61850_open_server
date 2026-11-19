package recovery

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

// A4: an unreadable journal is moved (not deleted) into the recovery folder with a copy of
// the autosave; the autosave itself stays for startup recovery.
func TestQuarantineJournalMovesTheJournalAndKeepsTheAutosave(t *testing.T) {
	dir := testutil.TempDir(t)
	journal, saved := []byte("{oops\n{oops\n"), []byte("saved project")
	if err := os.WriteFile(filepath.Join(dir, "autosave.journal"), journal, 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "autosave.scadabench"), saved, 0o666); err != nil {
		t.Fatal(err)
	}
	notice, err := QuarantineJournal(dir, errors.New("autosave journal line 1 is damaged"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "autosave.journal")); !os.IsNotExist(err) {
		t.Fatal("the journal must leave the data folder")
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "autosave.scadabench")); string(data) != string(saved) {
		t.Fatal("the autosave must stay in place")
	}
	if filepath.Dir(notice.backup) != filepath.Join(dir, FolderName) || !strings.HasSuffix(notice.backup, ".journal") {
		t.Fatalf("backup %s", notice.backup)
	}
	if data, _ := os.ReadFile(notice.backup); string(data) != string(journal) {
		t.Fatal("the journal bytes must be kept")
	}
	copied := strings.TrimSuffix(notice.backup, ".journal") + ".scadabench"
	if data, _ := os.ReadFile(copied); string(data) != string(saved) {
		t.Fatal("the autosave copy must sit next to the journal")
	}

	empty := notice.Apply(nil)
	restored := notice.Apply(pyjson.NewObject().Set("status", "restored").Set("message", "Recovered"))
	for _, result := range []*pyjson.Object{empty, restored} {
		if status, _ := result.Get("status"); status != "unreadable" {
			t.Fatalf("status %v", status)
		}
		if backup, _ := result.Get("backup"); backup != notice.backup {
			t.Fatalf("backup %v", backup)
		}
	}
	bad := pyjson.NewObject().Set("status", "unreadable").Set("message", "autosave")
	if notice.Apply(bad) != bad {
		t.Fatal("an unreadable autosave keeps its own notice")
	}
}
