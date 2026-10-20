package service

import (
	"slices"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// updates records the "update" batches a fake studio_gateway receives.
type updates struct {
	mu      sync.Mutex
	batches [][]any
}

func (u *updates) Request(op string, _ time.Duration, fields *pyjson.Object) (any, error) {
	if op == "update" {
		values, _ := fields.Get("values")
		u.mu.Lock()
		u.batches = append(u.batches, values.([]any))
		u.mu.Unlock()
	}
	return pyjson.NewObject(), nil
}

func (u *updates) Alive() bool { return true }
func (u *updates) Close()      {}

func (u *updates) count() int {
	u.mu.Lock()
	defer u.mu.Unlock()
	return len(u.batches)
}

func (u *updates) waitFor(t *testing.T, n int) []any {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for u.count() < n {
		if time.Now().After(deadline) {
			t.Fatalf("waited for update %d, got %d", n, u.count())
		}
		time.Sleep(5 * time.Millisecond)
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.batches[n-1]
}

// names records OnChange calls.
type names struct {
	mu   sync.Mutex
	seen []string
}

func (n *names) add(name string) {
	n.mu.Lock()
	n.seen = append(n.seen, name)
	n.mu.Unlock()
}

func (n *names) has(name string) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return slices.Contains(n.seen, name)
}

// TestProjectEditReachesGatewayWithoutSweep: an IED role edit the value hub does not report
// is pushed on ProjectChanged (the project_changed event); no periodic sweep runs meanwhile.
func TestProjectEditReachesGatewayWithoutSweep(t *testing.T) {
	hub := newHub()
	hub.define("bay", "GGIO1.Ind1.stVal", Signal{Value: true})
	ied := pyjson.NewObject().Set("id", "bay").Set("name", "Bay").Set("role", "simulated").Set("state", "running")
	projects := &fakeProjects{ieds: []*pyjson.Object{ied}}
	manager := NewManager(projects, hub, nil, "", nil)
	seen := &names{}
	manager.OnChange(seen.add)
	process := &updates{}
	point := pyjson.NewObject().Set("id", "p1").Set("kind", "single").Set("ied", "bay").Set("ref", "GGIO1.Ind1.stVal")
	sync := newSyncer(manager, process, []*pyjson.Object{point})
	manager.mu.Lock()
	manager.sync = sync
	manager.mu.Unlock()
	sync.start()
	defer sync.close()
	first := process.waitFor(t, 1)[0].(*pyjson.Object)
	if online, _ := first.Get("source_ok"); online != true {
		t.Fatalf("first batch: %v", first)
	}
	projects.mu.Lock()
	projects.ieds = []*pyjson.Object{copyObject(ied).Set("role", "real")} // a real IED that is not live
	projects.mu.Unlock()
	time.Sleep(3 * MinGap)
	if process.count() != 1 {
		t.Fatal("nothing re-checks the points by itself (no sweep timer)")
	}
	manager.ProjectChanged()
	second := process.waitFor(t, 2)[0].(*pyjson.Object)
	if online, _ := second.Get("source_ok"); online != false {
		t.Fatalf("the role edit must reach SCADA: %v", second)
	}
	if !seen.has(ChangeGatewayPoints) {
		t.Fatalf("ProjectChanged reports gateway.points: %v", seen.seen)
	}
}

func TestLogsAndStateReportChanges(t *testing.T) {
	manager := NewManager(&fakeProjects{}, newHub(), nil, "", nil)
	gateway := &names{}
	manager.OnChange(gateway.add)
	manager.Traffic.Add(pyjson.NewObject().Set("dir", "info"))
	seq, _ := manager.Commands.Add(pyjson.NewObject()).Get("seq")
	manager.Commands.Update(seq.(int64), pyjson.NewObject().Set("applied", "x"))
	manager.event(pyjson.NewObject().Set("event", "gw_client").Set("peer", "1.2.3.4").Set("state", "open"))
	for _, name := range []string{ChangeGatewayTraffic, ChangeGatewayStatus, ChangeGatewayCommands} {
		if !gateway.has(name) {
			t.Errorf("gateway did not report %s: %v", name, gateway.seen)
		}
	}
	master := NewMaster(&fakeProjects{}, nil, "", nil)
	reported := &names{}
	master.OnChange(reported.add)
	master.event(pyjson.NewObject().Set("event", "master_values").Set("values", []any{}))
	master.Traffic.Add(info("x"))
	master.Commands.Add(pyjson.NewObject())
	for _, name := range []string{ChangeMasterStatus, ChangeMasterPoints, ChangeMasterTraffic, ChangeMasterCommands} {
		if !reported.has(name) {
			t.Errorf("master did not report %s: %v", name, reported.seen)
		}
	}
	manager.OnChange(nil)
	manager.Traffic.Add(pyjson.NewObject())
}
