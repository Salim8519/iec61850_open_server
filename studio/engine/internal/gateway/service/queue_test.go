package service

import (
	"errors"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// fakeGatewayProcess records the answers studio_gateway would get; halt fails when haltErr is set.
type fakeGatewayProcess struct {
	mu      sync.Mutex
	answers []*pyjson.Object
	haltErr error
}

func (p *fakeGatewayProcess) Request(op string, _ time.Duration, fields *pyjson.Object) (any, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if op == "command_result" {
		p.answers = append(p.answers, fields)
	}
	if op == "halt" {
		return nil, p.haltErr
	}
	return nil, nil
}
func (p *fakeGatewayProcess) Alive() bool { return true }
func (p *fakeGatewayProcess) Close()      {}

func (p *fakeGatewayProcess) results() []*pyjson.Object {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]*pyjson.Object(nil), p.answers...)
}

// slowOperator holds every operate until release is closed.
type slowOperator struct {
	mu      sync.Mutex
	calls   int
	started chan struct{}
	release chan struct{}
}

func (o *slowOperator) Operate(string, any, string, any, any) (*pyjson.Object, error) {
	o.mu.Lock()
	o.calls++
	o.mu.Unlock()
	o.started <- struct{}{}
	<-o.release
	return pyjson.NewObject().Set("ok", true), nil
}

func runningGateway(t *testing.T, operator *slowOperator, process *fakeGatewayProcess) *Manager {
	t.Helper()
	projects := &fakeProjects{ieds: []*pyjson.Object{bay()}, section: gateway.EmptyGateway()}
	manager := NewManager(projects, newHub(), operator, "", nil)
	plannedBay(t, projects)
	manager.mu.Lock()
	manager.process, manager.state = process, "running"
	manager.mu.Unlock()
	return manager
}

func scadaCommand(token string) *pyjson.Object {
	return pyjson.NewObject().Set("event", "gw_command").Set("point_id", "bay:BayCTRL/XCBR1.Pos:command_double").
		Set("value", true).Set("protocol", "iec104").Set("peer", "master").Set("token", token)
}

// Stop answers every waiting SCADA command "refused" and sends none of them; only the one
// already running reaches the IED (fix plan A7).
func TestStopRefusesWaitingCommands(t *testing.T) {
	operator := &slowOperator{started: make(chan struct{}, 8), release: make(chan struct{})}
	process := &fakeGatewayProcess{}
	manager := runningGateway(t, operator, process)
	for _, token := range []string{"t1", "t2", "t3", "t4"} {
		manager.event(scadaCommand(token))
	}
	<-operator.started
	if _, err := manager.Stop(false); err != nil {
		t.Fatal(err)
	}
	close(operator.release)
	deadline := time.Now().Add(2 * time.Second)
	for len(process.results()) < 4 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	operator.mu.Lock()
	calls := operator.calls
	operator.mu.Unlock()
	if calls != 1 {
		t.Fatalf("%d commands reached the IED after Stop, want only the running one", calls)
	}
	refused := 0
	for _, result := range process.results() {
		if get(result, "ok") == false && get(result, "reason") == reasonStopped {
			refused++
		}
	}
	if refused != 3 {
		t.Fatalf("answers %v, want three refused with %q", process.results(), reasonStopped)
	}
}

// A full queue refuses the command at once instead of running it later out of order.
func TestFullQueueRefuses(t *testing.T) {
	process := &fakeGatewayProcess{}
	manager := NewManager(&fakeProjects{section: gateway.EmptyGateway()}, newHub(), nil, "", nil)
	manager.executor = &executor{manager: manager, jobs: make(chan job)} // no room, no loop
	manager.executor.submit(process, scadaCommand("t9"))
	deadline := time.Now().Add(2 * time.Second)
	for len(process.results()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if results := process.results(); len(results) != 1 || get(results[0], "reason") != reasonQueueFull {
		t.Fatalf("answers %v", results)
	}
}

// A halt that fails still leaves the gateway stopped: the process is gone (fix plan A11).
func TestFailedHaltIsNotRunning(t *testing.T) {
	operator := &slowOperator{started: make(chan struct{}, 1), release: make(chan struct{})}
	manager := runningGateway(t, operator, &fakeGatewayProcess{haltErr: errors.New("Native halt request timed out")})
	if _, err := manager.Stop(false); err == nil {
		t.Fatal("the halt error is reported")
	}
	if state := manager.State(); state == "running" {
		t.Fatalf("state %q with no gateway process", state)
	}
}
