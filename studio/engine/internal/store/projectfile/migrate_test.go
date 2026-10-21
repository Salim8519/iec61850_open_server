package projectfile

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"studio-engine/internal/testutil"
)

func put(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func journalLines(t *testing.T, dir string) []map[string]string {
	t.Helper()
	var lines []map[string]string
	for _, line := range strings.Split(strings.TrimSpace(read(t, filepath.Join(dir, "logs", JournalName))), "\n") {
		entry := map[string]string{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("journal line %q: %v", line, err)
		}
		lines = append(lines, entry)
	}
	return lines
}

func kinds(records []Record) []string {
	var found []string
	for _, record := range records {
		found = append(found, record.Kind)
	}
	return found
}

func TestNothingToMigrateWritesNothing(t *testing.T) {
	dir := testutil.TempDir(t)
	records, err := Migrate(dir)
	if err != nil || len(records) != 0 {
		t.Fatalf("records %v, error %v", records, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "logs")); !os.IsNotExist(err) {
		t.Fatal("a folder without old files must not get a journal")
	}
}

func TestAutosaveAndRecoveryFilesAreRenamedWithTheirBytes(t *testing.T) {
	dir := testutil.TempDir(t)
	put(t, filepath.Join(dir, "autosave.subsim"), "project")
	put(t, filepath.Join(dir, "recovery", "unreadable-autosave-ab.SUBSIM"), "damaged")
	put(t, filepath.Join(dir, "recovery", "notes.txt"), "leave me")
	records, err := Migrate(dir)
	if err != nil || strings.Join(kinds(records), ",") != KindRenamed+","+KindRenamed {
		t.Fatalf("records %v, error %v", records, err)
	}
	if read(t, filepath.Join(dir, AutosaveName)) != "project" ||
		read(t, filepath.Join(dir, "recovery", "unreadable-autosave-ab.scadabench")) != "damaged" ||
		read(t, filepath.Join(dir, "recovery", "notes.txt")) != "leave me" {
		t.Fatal("renamed files must keep their bytes")
	}
	for _, old := range []string{"autosave.subsim", filepath.Join("recovery", "unreadable-autosave-ab.SUBSIM")} {
		if _, err := os.Stat(filepath.Join(dir, old)); !os.IsNotExist(err) {
			t.Fatalf("%s must be gone after the rename", old)
		}
	}
	lines := journalLines(t, dir)
	if len(lines) != 2 || lines[0]["from"] != "autosave.subsim" || lines[0]["to"] != "autosave.scadabench" ||
		lines[1]["to"] != "recovery/unreadable-autosave-ab.scadabench" || lines[0]["correlation_id"] != lines[1]["correlation_id"] {
		t.Fatalf("journal %v", lines)
	}
}

func TestRunningTwiceChangesNothingTheSecondTime(t *testing.T) {
	dir := testutil.TempDir(t)
	put(t, filepath.Join(dir, "autosave.subsim"), "project")
	if _, err := Migrate(dir); err != nil {
		t.Fatal(err)
	}
	again, err := Migrate(dir)
	if err != nil || len(again) != 0 || len(journalLines(t, dir)) != 1 {
		t.Fatalf("second run: %v %v", again, err)
	}
}

func TestAnExistingNewNameIsNeverOverwritten(t *testing.T) {
	dir := testutil.TempDir(t)
	put(t, filepath.Join(dir, "autosave.subsim"), "old")
	put(t, filepath.Join(dir, "autosave.scadabench"), "new")
	put(t, filepath.Join(dir, "recovery", "a.subsim"), "older backup")
	put(t, filepath.Join(dir, "recovery", "a.scadabench"), "newer backup")
	records, err := Migrate(dir)
	if err != nil || strings.Join(kinds(records), ",") != KindKeptBoth+","+KindKeptBoth {
		t.Fatalf("records %v, error %v", records, err)
	}
	want := map[string]string{"autosave.subsim": "old", "autosave.scadabench": "new",
		filepath.Join("recovery", "a.subsim"): "older backup", filepath.Join("recovery", "a.scadabench"): "newer backup"}
	for name, text := range want {
		if read(t, filepath.Join(dir, name)) != text {
			t.Fatalf("%s changed", name)
		}
	}
}

func TestAFolderWithTheOldNameIsLeftAlone(t *testing.T) {
	dir := testutil.TempDir(t)
	if err := os.Mkdir(filepath.Join(dir, "autosave.subsim"), 0o755); err != nil {
		t.Fatal(err)
	}
	records, err := Migrate(dir)
	if err != nil || strings.Join(kinds(records), ",") != KindLeftAlone {
		t.Fatalf("records %v, error %v", records, err)
	}
	if info, err := os.Stat(filepath.Join(dir, "autosave.subsim")); err != nil || !info.IsDir() {
		t.Fatal("the folder must stay")
	}
}

func TestEverySortOfFileIsJournaledInNameOrder(t *testing.T) {
	dir := testutil.TempDir(t)
	for _, name := range []string{"b.subsim", "a.subsim", "c.SubSim"} {
		put(t, filepath.Join(dir, "recovery", name), name)
	}
	records, _ := Migrate(dir)
	var to []string
	for _, record := range records {
		to = append(to, record.To)
	}
	if !sort.StringsAreSorted(to) || len(to) != 3 {
		t.Fatalf("records %v", to)
	}
}

func TestUniqueNamesAreUuidHexWithTheNewExtension(t *testing.T) {
	name := UniqueName("before-ied-delete-")
	if !strings.HasPrefix(name, "before-ied-delete-") || !strings.HasSuffix(name, ".scadabench") ||
		len(name) != len("before-ied-delete-")+32+len(".scadabench") || name == UniqueName("before-ied-delete-") {
		t.Fatalf("name %q", name)
	}
}

func TestOnlyTheOldExtensionIsOld(t *testing.T) {
	for name, want := range map[string]bool{"a.subsim": true, "A.SUBSIM": true, `C:\p\station.SubSim`: true,
		"a.scadabench": false, "a.zip": false, "subsim": false, "": false} {
		if IsOldName(name) != want {
			t.Errorf("IsOldName(%q) != %v", name, want)
		}
	}
	if !bytes.Contains([]byte(OldProjectMessage), []byte("(.subsim)")) {
		t.Fatal("the refusal names the old extension")
	}
}
