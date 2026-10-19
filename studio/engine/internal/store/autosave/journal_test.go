package autosave

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/apperr"
	"studio-engine/internal/store/archive"
)

func TestWriteAutosaveReplacesThroughTemp(t *testing.T) {
	dir := t.TempDir()
	saveDoc(t, dir, doc("one"))
	saveDoc(t, dir, doc("two"))
	assertAutosave(t, dir, doc("two"))
	if exists(filepath.Join(dir, TempName)) {
		t.Fatal("autosave.tmp left behind")
	}
}

func TestJournalFoldsEveryCommitAfterACrash(t *testing.T) {
	dir := t.TempDir()
	scl := strings.Repeat("<SCL big=\"1\"/>\n", 1000)
	base := doc("base", scl)
	saveDoc(t, dir, base)
	journal, err := openTest(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	first := derive(base, "diagram", mustGet(doc("first"), "diagram"))
	second := derive(first, "name", "Renamed", "gateway", true)
	added := doc("x", scl, "<SCL>new</SCL>")
	third := derive(second, "ieds", mustGet(added, "ieds"), "scl_sources", mustGet(added, "scl_sources"))
	for rev, pair := range [][2]any{{base, first}, {first, second}, {second, third}} {
		if err := journal.Commit(uint64(rev+1), pair[0].(*pyObject), pair[1].(*pyObject)); err != nil {
			t.Fatal(err)
		}
	}
	lines := journalLines(t, dir)
	if len(lines) != 4 || strings.Contains(lines[1], "big") || strings.Contains(lines[2], "big") ||
		strings.Contains(lines[3], "big") || !strings.Contains(lines[3], "<SCL>new</SCL>") {
		t.Fatalf("journal records carry unchanged SCL text:\n%s", strings.Join(lines, "\n"))
	}
	crash(journal) // the process dies here: no snapshot was written; the next start folds
	applied, err := Fold(dir, nil)
	if err != nil || applied != 3 {
		t.Fatalf("fold applied %d: %v", applied, err)
	}
	assertAutosave(t, dir, third)
	if exists(filepath.Join(dir, JournalName)) {
		t.Fatal("journal not removed after fold")
	}
}

func TestFirstCommitIsFullUnlessAutosaveMatches(t *testing.T) {
	dir := t.TempDir()
	journal, _ := openTest(t, dir)
	if err := journal.Commit(1, doc("a"), doc("b")); err != nil { // no autosave on disk
		t.Fatal(err)
	}
	if lines := journalLines(t, dir); !strings.Contains(lines[1], `"doc":`) || !strings.Contains(lines[0], `"base":""`) {
		t.Fatalf("expected a full record: %v", lines)
	}
	crash(journal)
	if applied, err := Fold(dir, nil); err != nil || applied != 1 {
		t.Fatalf("fold: %d %v", applied, err)
	}
	assertAutosave(t, dir, doc("b"))
	journal, _ = openTest(t, dir)
	if err := journal.Commit(2, doc("b"), doc("c")); err != nil { // autosave holds "b" exactly
		t.Fatal(err)
	}
	if lines := journalLines(t, dir); strings.Contains(lines[1], `"doc":`) {
		t.Fatalf("expected a change record: %v", lines)
	}
}

func TestFoldTornAndDamagedLines(t *testing.T) {
	dir := t.TempDir()
	saveDoc(t, dir, doc("a"))
	journal, _ := openTest(t, dir)
	_ = journal.Commit(1, doc("a"), doc("b"))
	crash(journal)
	path := filepath.Join(dir, JournalName)
	file, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = file.WriteString(`{"rev":2,"keys":["id"`)
	_ = file.Close()
	if applied, err := Fold(dir, nil); err != nil || applied != 1 {
		t.Fatalf("torn last line: %d %v", applied, err)
	}
	assertAutosave(t, dir, doc("b"))

	journal, _ = openTest(t, dir)
	_ = journal.Commit(2, doc("b"), doc("c"))
	_ = journal.Commit(3, doc("c"), doc("d"))
	lines := journalLines(t, dir)
	damaged := lines[0] + "\n" + "{oops\n" + lines[2] + "\n"
	crash(journal)
	_ = os.WriteFile(path, []byte(damaged), 0o666)
	if _, err := Fold(dir, nil); err == nil || !strings.Contains(err.Error(), "line 2 is damaged") {
		t.Fatalf("damaged middle line: %v", err)
	}
	if !exists(path) {
		t.Fatal("journal removed after an error")
	}
	assertAutosave(t, dir, doc("b"))
}

func TestFoldRefusesAForeignAutosave(t *testing.T) {
	dir := t.TempDir()
	saveDoc(t, dir, doc("a"))
	journal, _ := openTest(t, dir)
	_ = journal.Commit(1, doc("a"), doc("b"))
	crash(journal)
	saveDoc(t, dir, doc("python wrote this"))
	if _, err := Fold(dir, nil); !errors.Is(err, ErrJournalMismatch) {
		t.Fatalf("expected mismatch, got %v", err)
	}
	if _, err := OpenJournal(dir, nil); !errors.Is(err, ErrNotFolded) {
		t.Fatalf("open with leftover journal: %v", err)
	}
}

func TestCommitRefusesNonFiniteAndOldRevisions(t *testing.T) {
	dir := t.TempDir()
	journal, _ := openTest(t, dir)
	err := journal.Commit(1, nil, derive(doc("a"), "bad", math.Inf(-1)))
	if appErr, ok := err.(*apperr.Error); !ok || appErr.Message != "Out of range float values are not JSON compliant: -inf" {
		t.Fatalf("non-finite: %v", err)
	}
	if err := journal.Commit(5, nil, doc("a")); err != nil {
		t.Fatal(err)
	}
	if err := journal.Commit(5, doc("a"), doc("b")); err == nil {
		t.Fatal("repeated revision accepted")
	}
	crash(journal)
}

func TestFoldSkipsRecordsAlreadyInAMarkedSnapshot(t *testing.T) {
	dir := t.TempDir()
	scl := "<SCL>one</SCL>"
	saveDoc(t, dir, doc("0", scl))
	journal, _ := openTest(t, dir)
	docs := []*pyObject{doc("0", scl)}
	for rev := 1; rev <= 3; rev++ {
		next := doc(string(rune('0'+rev)), scl)
		if rev == 2 { // revision 2 removes the source, revision 3 brings it back
			next = doc("2")
		}
		_ = journal.Commit(uint64(rev), docs[rev-1], next)
		docs = append(docs, next)
	}
	// Crash between "rename snapshot 2 in" and "trim the journal".
	payload, _ := archive.Encode(docs[2])
	if err := journal.markSnapshot(2, digest(payload)); err != nil {
		t.Fatal(err)
	}
	_ = WriteAutosave(dir, payload)
	crash(journal)
	if applied, err := Fold(dir, nil); err != nil || applied != 1 {
		t.Fatalf("fold after marked snapshot: %d %v", applied, err)
	}
	assertAutosave(t, dir, docs[3])
}
