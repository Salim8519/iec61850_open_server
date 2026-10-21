package autosave

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/store/archive"
)

func TestSnapshotWriterCoalescesAndKeepsTheLatest(t *testing.T) {
	dir := tempDir(t)
	writer := NewSnapshotWriter(dir, nil, nil, nil)
	for rev := 1; rev <= 50; rev++ {
		writer.Request(uint64(rev), doc(fmt.Sprint(rev), "<SCL/>"))
	}
	writer.Request(10, doc("stale")) // older than the latest request: ignored
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}
	assertAutosave(t, dir, doc("50", "<SCL/>"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	writer.Request(51, doc("after close"))
	assertAutosave(t, dir, doc("50", "<SCL/>"))
}

func TestSnapshotWriterReportsErrors(t *testing.T) {
	missing := filepath.Join(tempDir(t), "missing")
	var mu sync.Mutex
	var reported []uint64
	writer := NewSnapshotWriter(missing, nil, nil, func(rev uint64, err error) {
		mu.Lock()
		defer mu.Unlock()
		reported = append(reported, rev)
	})
	writer.Request(7, doc("x"))
	if err := writer.Flush(); err == nil {
		t.Fatal("expected the write error")
	}
	_ = writer.Close()
	mu.Lock()
	defer mu.Unlock()
	if len(reported) != 1 || reported[0] != 7 {
		t.Fatalf("reported %v", reported)
	}
}

func TestSnapshotTrimsJournalAndCloseRemovesIt(t *testing.T) {
	dir := tempDir(t)
	encoder := archive.NewEncoder()
	journal, _ := OpenJournal(dir, encoder)
	writer := NewSnapshotWriter(dir, encoder, journal, nil)
	previous := doc("0", "<SCL>a</SCL>")
	for rev := 1; rev <= 5; rev++ {
		next := derive(previous, "name", fmt.Sprint("name ", rev))
		if err := journal.Commit(uint64(rev), previous, next); err != nil {
			t.Fatal(err)
		}
		previous = next
		if rev == 3 {
			writer.Request(3, next)
			if err := writer.Flush(); err != nil {
				t.Fatal(err)
			}
		}
	}
	lines := journalLines(t, dir)
	payload, _ := os.ReadFile(filepath.Join(dir, AutosaveName))
	if len(lines) != 3 || !strings.Contains(lines[0], digest(payload)) ||
		!strings.HasPrefix(lines[1], `{"rev":4`) || !strings.HasPrefix(lines[2], `{"rev":5`) {
		t.Fatalf("journal after snapshot 3:\n%s", strings.Join(lines, "\n"))
	}
	writer.Request(5, previous)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if exists(filepath.Join(dir, JournalName)) {
		t.Fatal("journal kept after the final snapshot")
	}
	assertAutosave(t, dir, previous)
}

func TestCrashAfterPartialSnapshotFoldsTheRest(t *testing.T) {
	dir := tempDir(t)
	encoder := archive.NewEncoder()
	journal, _ := openTest(t, dir)
	writer := NewSnapshotWriter(dir, encoder, journal, nil)
	previous := doc("0", "<SCL>a</SCL>")
	for rev := 1; rev <= 6; rev++ {
		next := derive(previous, "diagram", mustGet(doc(fmt.Sprint(rev)), "diagram"))
		if rev == 4 {
			next = doc("4", "<SCL>a</SCL>", "<SCL>b</SCL>") // a source is added
		}
		if err := journal.Commit(uint64(rev), previous, next); err != nil {
			t.Fatal(err)
		}
		previous = next
		if rev == 2 || rev == 5 {
			writer.Request(uint64(rev), next)
			_ = writer.Flush()
		}
	}
	crash(journal) // dies before snapshot 6
	if applied, err := Fold(dir, nil); err != nil || applied != 1 {
		t.Fatalf("fold: %d %v", applied, err)
	}
	assertAutosave(t, dir, previous)
}
