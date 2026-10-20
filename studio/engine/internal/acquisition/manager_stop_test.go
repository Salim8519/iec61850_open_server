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

// slowWorker blocks every request until Close, except "disconnect", which takes delay
// (a live client giving back its report blocks).
type slowWorker struct {
	delay  time.Duration
	closed chan struct{}
	once   sync.Once
}

func (w *slowWorker) Request(op string, _ time.Duration, _ *pyjson.Object) (any, error) {
	if op == "disconnect" {
		time.Sleep(w.delay)
		return nil, nil
	}
	<-w.closed
	return nil, errors.New("closed")
}

func (w *slowWorker) Close() { w.once.Do(func() { close(w.closed) }) }

// TestStopClosesSessionsTogetherInOrder (gap 2): Stop closes the sessions' live clients at
// the same time, then emits their "stopped" live_status events in connection order, as
// Python's sequential stop did.
func TestStopClosesSessionsTogetherInOrder(t *testing.T) {
	live := filepath.Join(testutil.TempDir(t), "studio_live.exe")
	if err := os.WriteFile(live, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	const delay = 300 * time.Millisecond
	var ieds []any
	for _, id := range []string{"real1", "real2", "real3"} {
		ieds = append(ieds, pyv.O("id", id, "name", strings.ToUpper(id), "role", "real",
			"live", pyv.O("ip", "127.67.9.1", "port", int64(20102), "access", "auto_reports", "poll_ms", int64(1000)),
			"signals", []any{}))
	}
	project := pyv.O("ieds", ieds, "diagram", pyv.O("nodes", []any{}), "test_panel", []any{},
		"network", pyv.O("mode", "loopback"))
	events := &testEvents{}
	start := func(string, func(*pyjson.Object)) (Worker, error) {
		return &slowWorker{delay: delay, closed: make(chan struct{})}, nil
	}
	manager := NewManager(Deps{LiveExecutable: live, Start: start, Projects: testProjects{project},
		Runtime: testRuntime{"running"}, Events: events, Bus: events, Values: newTestValues(project),
		Samples: testIdle{}, Equipment: testIdle{}, States: testIdle{}, Network: testIdle{},
		Clock: testClock{start: time.Now()}})
	if err := manager.Connect(project, pyv.Objects(ieds)); err != nil {
		t.Fatal(err)
	}
	waitFor(t, 5*time.Second, func() bool {
		connecting := 0
		for _, event := range events.all() {
			if strings.Contains(event, `"state": "connecting"`) {
				connecting++
			}
		}
		return connecting == len(ieds)
	})
	began := time.Now()
	manager.Stop()
	elapsed := time.Since(began)
	if elapsed >= 2*delay {
		t.Fatalf("Stop took %v: the live clients were closed one after another", elapsed)
	}
	all := events.all()
	var stopped []string
	for _, event := range all {
		if strings.Contains(event, `"state": "stopped"`) {
			stopped = append(stopped, event[strings.Index(event, `"ied_id": "`)+11:][:5])
		}
	}
	if strings.Join(stopped, ",") != "real1,real2,real3" {
		t.Fatalf("stopped events out of connection order: %v\n%s", stopped, strings.Join(all, "\n"))
	}
	if !strings.Contains(all[len(all)-1], `"real3"`) || !strings.Contains(all[len(all)-1], `"stopped"`) {
		t.Fatalf("the stopped events must come last: %s", strings.Join(all, "\n"))
	}
	t.Logf("3 sessions stopped in %v (each disconnect %v)", elapsed, delay)
}
