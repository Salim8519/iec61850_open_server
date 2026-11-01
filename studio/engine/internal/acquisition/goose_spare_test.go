package acquisition

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/acquisition/goosedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

// engineWorker answers hello and open like studio_goose and records whether it was closed.
type engineWorker struct {
	mu     sync.Mutex
	ops    []string
	closed bool
}

func (w *engineWorker) Request(op string, _ time.Duration, _ *pyjson.Object) (any, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.ops = append(w.ops, op)
	capabilities := []any{}
	for _, capability := range goosedom.EngineCapabilities() {
		capabilities = append(capabilities, capability)
	}
	if op == "hello" {
		return pyv.O("capabilities", capabilities), nil
	}
	return pyv.O("port", int64(50000), "sources", pyjson.NewObject()), nil
}

func (w *engineWorker) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
}

func (w *engineWorker) isClosed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.closed
}

type engineStarts struct {
	mu      sync.Mutex
	started []*engineWorker
}

func (s *engineStarts) start(string, func(*pyjson.Object)) (Worker, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	worker := &engineWorker{}
	s.started = append(s.started, worker)
	return worker, nil
}

func (s *engineStarts) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.started)
}

func spareTestEngine(t *testing.T) (*GooseEngine, *engineStarts, string) {
	executable := filepath.Join(testutil.TempDir(t), "studio_goose.exe")
	if err := os.WriteFile(executable, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	starts := &engineStarts{}
	return NewGooseEngine(Deps{Start: starts.start}, executable), starts, executable
}

// A Run takes the spare (no new process); the next spare starts only when asked again.
func TestRunUsesTheSpareEngine(t *testing.T) {
	engine, starts, _ := spareTestEngine(t)
	engine.Prepare()
	if err := engine.Ensure(); err != nil {
		t.Fatal(err)
	}
	if starts.count() != 1 || !engine.Running() {
		t.Fatalf("processes started %d, running %v; want the spare only", starts.count(), engine.Running())
	}
	engine.Prepare() // a Run is on: no spare beside the running engine
	engine.Stop()
	if starts.count() != 1 || !starts.started[0].isClosed() {
		t.Fatalf("after Stop: %d processes, first closed %v", starts.count(), starts.started[0].isClosed())
	}
}

// A spare started before the program file changed is closed; the Run starts the new file.
func TestChangedProgramIsNeverUsedStale(t *testing.T) {
	engine, starts, executable := spareTestEngine(t)
	engine.Prepare()
	engine.takeSpareForTest(t, starts)
	later := time.Now().Add(2 * time.Second)
	if err := os.WriteFile(executable, []byte("v2 rebuilt"), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.Chtimes(executable, later, later)
	if err := engine.Ensure(); err != nil {
		t.Fatal(err)
	}
	if starts.count() != 2 || !starts.started[0].isClosed() || starts.started[1].isClosed() {
		t.Fatalf("want the stale spare closed and a fresh engine running: %d started", starts.count())
	}
	engine.Stop()
}

func TestDiscardLeavesNoSpare(t *testing.T) {
	engine, starts, _ := spareTestEngine(t)
	engine.Prepare()
	engine.Discard()
	engine.Prepare() // shutting down: nothing new starts
	if starts.count() != 1 || !starts.started[0].isClosed() {
		t.Fatalf("after Discard: %d started, first closed %v", starts.count(), starts.started[0].isClosed())
	}
}

// takeSpareForTest waits until the background spare has started (and leaves it in place).
func (e *GooseEngine) takeSpareForTest(t *testing.T, starts *engineStarts) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		e.mu.Lock()
		ready := e.spare.process != nil
		e.mu.Unlock()
		if ready {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("no spare after 5 s (%d started)", starts.count())
}
