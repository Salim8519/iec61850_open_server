package realcmd

import (
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/worker"
)

// heldControl answers only when release is closed, or times out like studio_control.
type heldControl struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
	timeout bool
}

func (h *heldControl) Operate(string, any, string, any, any) (*pyjson.Object, error) {
	h.mu.Lock()
	h.calls++
	h.mu.Unlock()
	if h.timeout {
		return nil, &worker.Timeout{Op: "operate"}
	}
	h.started <- struct{}{}
	<-h.release
	return pyjson.NewObject().Set("ok", true), nil
}

func heldService(operator *heldControl, events *[]string, mu *sync.Mutex) *Service {
	session := &fakeSession{state: "live", values: []any{int64(2)}}
	return &Service{FindIED: func(string) (*pyjson.Object, error) { return breaker(true), nil },
		Sessions: func(string) (Session, error) { return session, nil }, Control: operator,
		Emit: func(kind string, _ *pyjson.Object) {
			mu.Lock()
			*events = append(*events, kind)
			mu.Unlock()
		}, Observe: 50 * time.Millisecond, Poll: time.Millisecond}
}

// A second command to the same IED and reference while the first runs is refused, never
// queued behind it (fix plan A8).
func TestSecondOperateWhileOneRunsIsRefused(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	operator := &heldControl{started: make(chan struct{}, 2), release: make(chan struct{})}
	service := heldService(operator, &events, &mu)
	done := make(chan error, 1)
	go func() { _, err := service.Operate("r1", "LD/CSWI1.Pos", true, true); done <- err }()
	<-operator.started
	_, err := service.Operate("r1", "LD/CSWI1.Pos", true, true)
	if err == nil || err.Error() != "A command to LD/CSWI1.Pos on Bay1 is still running; wait for its result" {
		t.Fatalf("second command: %v", err)
	}
	close(operator.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if operator.calls != 1 {
		t.Fatalf("%d commands sent, want 1", operator.calls)
	}
	if _, err := service.Operate("r1", "LD/CSWI1.Pos", false, true); err != nil {
		t.Fatalf("a command after the first finished is sent: %v", err)
	}
}

// No answer in time is not a failure: the result is unknown and the user is told to check
// the position first.
func TestTimeoutIsResultUnknown(t *testing.T) {
	var mu sync.Mutex
	events := []string{}
	service := heldService(&heldControl{timeout: true}, &events, &mu)
	_, err := service.Operate("r1", "LD/CSWI1.Pos", true, true)
	want := "Bay1 did not answer in time; result unknown — check the position before trying again"
	if err == nil || err.Error() != want {
		t.Fatalf("error %v, want %q", err, want)
	}
	if got := strings.Join(events, ","); got != "real_command_requested,real_command_unknown" {
		t.Fatalf("events %s", got)
	}
}
