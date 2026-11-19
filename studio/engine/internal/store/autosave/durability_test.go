package autosave

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/store/archive"
)

// recordFileSteps replaces the sync and rename seams with recorders of "sync <name>" and
// "rename <name>" steps (base names); rename runs through hook when given.
func recordFileSteps(t *testing.T, hook func(from, to string) error) func() []string {
	var mu sync.Mutex
	var steps []string
	add := func(step string) {
		mu.Lock()
		defer mu.Unlock()
		steps = append(steps, step)
	}
	t.Cleanup(func() { syncFile, renameFile = (*os.File).Sync, os.Rename })
	syncFile = func(file *os.File) error {
		add("sync " + filepath.Base(file.Name()))
		return file.Sync()
	}
	renameFile = func(from, to string) error {
		add("rename " + filepath.Base(from))
		if hook != nil {
			return hook(from, to)
		}
		return os.Rename(from, to)
	}
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), steps...)
	}
}

// A3: the autosave bytes are flushed to disk before autosave.tmp is renamed into place
// (and before the journal is trimmed), on the snapshot path and on WriteAutosave (Fold).
func TestAutosaveIsSyncedBeforeRename(t *testing.T) {
	dir := tempDir(t)
	steps := recordFileSteps(t, nil)
	encoder := archive.NewEncoder()
	journal, _ := openTest(t, dir)
	writer := NewSnapshotWriter(dir, encoder, journal, nil)
	if err := journal.Commit(1, nil, doc("a")); err != nil {
		t.Fatal(err)
	}
	writer.Request(1, doc("a"))
	if err := writer.Flush(); err != nil {
		t.Fatal(err)
	}
	saveDoc(t, dir, doc("b"))
	want := []string{"sync " + TempName, "rename " + TempName, "sync " + TempName, "rename " + TempName}
	got := steps()
	filtered := []string{}
	for _, step := range got {
		if step == "sync "+TempName || step == "rename "+TempName {
			filtered = append(filtered, step)
		}
	}
	if len(filtered) != len(want) {
		t.Fatalf("steps %v", got)
	}
	for index := range want {
		if filtered[index] != want[index] {
			t.Fatalf("autosave.tmp must be synced before it is renamed: %v", got)
		}
	}
	_ = writer.Close()
}

// A5: two overlapping Flush calls whose requests are covered by one failed attempt both
// report the failure (the earlier one used to return nil).
func TestOverlappingFlushesBothReportTheFailure(t *testing.T) {
	dir := tempDir(t)
	release := make(chan struct{})
	failed := errors.New("disk full")
	var calls int
	var mu sync.Mutex
	recordFileSteps(t, func(from, to string) error {
		mu.Lock()
		calls++
		first := calls == 1
		mu.Unlock()
		if first {
			<-release // rev 1 is being written while revs 2 and 3 are requested
			return os.Rename(from, to)
		}
		return failed
	})
	writer := NewSnapshotWriter(dir, nil, nil, nil)
	defer writer.Close()
	writer.Request(1, doc("1"))
	waitFor(t, func() bool { mu.Lock(); defer mu.Unlock(); return calls == 1 })
	results := make(chan error, 2)
	writer.Request(2, doc("2"))
	go func() { results <- writer.Flush() }()
	waitFor(t, func() bool { writer.mu.Lock(); defer writer.mu.Unlock(); return writer.urgent == 1 })
	writer.Request(3, doc("3")) // coalesces with 2: one attempt covers both
	go func() { results <- writer.Flush() }()
	waitFor(t, func() bool { writer.mu.Lock(); defer writer.mu.Unlock(); return writer.urgent == 2 })
	close(release)
	for index := 0; index < 2; index++ {
		if err := <-results; !errors.Is(err, failed) {
			t.Fatalf("flush %d returned %v, want the failed write", index+1, err)
		}
	}
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("condition not reached")
		}
		time.Sleep(time.Millisecond)
	}
}
