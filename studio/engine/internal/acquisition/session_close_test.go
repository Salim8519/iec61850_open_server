package acquisition

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/testutil"
)

// oneAtATime answers one request at a time like studio_live: "connect" never completes (a
// device that does not answer) unless connects is true; every other request waits for its turn.
type oneAtATime struct {
	connects bool
	slot     chan struct{}
	killed   chan struct{}
	once     sync.Once
	mu       sync.Mutex
	ops      []string
}

func newOneAtATime(connects bool) *oneAtATime {
	return &oneAtATime{connects: connects, slot: make(chan struct{}, 1), killed: make(chan struct{})}
}

var errGone = errors.New("worker exited")

func (w *oneAtATime) Request(op string, timeout time.Duration, _ *pyjson.Object) (any, error) {
	deadline := time.After(timeout)
	select {
	case w.slot <- struct{}{}:
	case <-w.killed:
		return nil, errGone
	case <-deadline:
		return nil, errors.New(op + " timed out")
	}
	defer func() { <-w.slot }()
	w.mu.Lock()
	w.ops = append(w.ops, op)
	w.mu.Unlock()
	switch {
	case op == "hello":
		return pyv.O("capabilities", stringsAny(liveCapabilities)), nil
	case op == "connect" && !w.connects:
		select {
		case <-w.killed:
			return nil, errGone
		case <-deadline:
			return nil, errors.New("connect timed out")
		}
	case op == "discover":
		return pyjson.NewObject(), nil
	case op == "rcb_status":
		return pyv.O("blocks", []any{pyv.O("rcb", "IED1LD0/LLN0.RP.urcb01", "readable", true, "free", true)}), nil
	case op == "rcb_reserve":
		return pyv.O("reserved", true), nil
	}
	return pyjson.NewObject(), nil
}

// Close is the native close: a "stop" request in turn (3 s), then the process ends.
func (w *oneAtATime) Close() {
	_, _ = w.Request("stop", 3*time.Second, nil)
	w.Kill()
}

func (w *oneAtATime) Kill() { w.once.Do(func() { close(w.killed) }) }

func (w *oneAtATime) sent() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.Join(w.ops, ",")
}

func stringsAny(items []string) []any {
	out := make([]any, len(items))
	for i, item := range items {
		out[i] = item
	}
	return out
}

func closeTestManager(t *testing.T, worker *oneAtATime) (*Manager, *pyjson.Object) {
	live := filepath.Join(testutil.TempDir(t), "studio_live.exe")
	if err := os.WriteFile(live, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	ied := pyv.O("id", "real1", "name", "REAL1", "role", "real", "signals", []any{},
		"reports", []any{pyv.O("ref", "IED1LD0/LLN0.RP.urcb01", "dataset_ref", "IED1LD0/LLN0.DS1")},
		"live", pyv.O("ip", "198.18.26.250", "port", int64(26102), "access", "auto_reports", "poll_ms", int64(1000)))
	project := pyv.O("ieds", []any{ied}, "diagram", pyv.O("nodes", []any{}), "test_panel", []any{},
		"network", pyv.O("mode", "loopback"))
	events := &testEvents{}
	start := func(string, func(*pyjson.Object)) (Worker, error) { return worker, nil }
	manager := NewManager(Deps{LiveExecutable: live, Start: start, Projects: testProjects{project},
		Runtime: testRuntime{"running"}, Events: events, Bus: events, Values: newTestValues(project),
		Samples: testIdle{}, Equipment: testIdle{}, States: testIdle{}, Network: testIdle{},
		Clock: testClock{start: time.Now()}})
	return manager, project
}

// TestStopWhileConnectingKillsTheClient (trial R6): Stop while the live client still waits
// for a device that does not answer took the 15 s connect timeout (the disconnect waited
// behind the connect); nothing is reserved yet, so the client is killed at once.
func TestStopWhileConnectingKillsTheClient(t *testing.T) {
	worker := newOneAtATime(false)
	manager, project := closeTestManager(t, worker)
	if err := manager.Connect(project, pyv.Objects(pyv.GetList(project, "ieds"))); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 5*time.Second, func() bool { return strings.Contains(worker.sent(), "connect") })
	began := time.Now()
	manager.Stop()
	if elapsed := time.Since(began); elapsed > 2*time.Second {
		t.Fatalf("Stop took %v while connecting (want under 2 s)", elapsed)
	}
	if strings.Contains(worker.sent(), "disconnect") {
		t.Fatalf("a killed client got a disconnect: %s", worker.sent())
	}
}

// TestStopAfterReservingStillDisconnects: once a report block is reserved, Stop sends
// "disconnect" (every block given back, ADR-0067) as before.
func TestStopAfterReservingStillDisconnects(t *testing.T) {
	worker := newOneAtATime(true)
	manager, project := closeTestManager(t, worker)
	if err := manager.Connect(project, pyv.Objects(pyv.GetList(project, "ieds"))); err != nil {
		t.Fatal(err)
	}
	session, _ := manager.Session("real1")
	waitFor(t, 5*time.Second, func() bool { return pyv.GetStr(session.Status(), "state") == "live" })
	manager.Stop()
	if sent := worker.sent(); !strings.Contains(sent, "rcb_reserve") || !strings.Contains(sent, "disconnect") {
		t.Fatalf("requests: %s (want a reserve, then a disconnect at Stop)", sent)
	}
}
