package service

import (
	"sync"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// cmdRecorder is a fake studio_master that accepts every command and remembers each request.
type cmdRecorder struct {
	mu   sync.Mutex
	sent []*pyjson.Object
}

func (r *cmdRecorder) Request(op string, _ time.Duration, fields *pyjson.Object) (any, error) {
	if op == "command" {
		r.mu.Lock()
		r.sent = append(r.sent, fields)
		r.mu.Unlock()
	}
	return pyjson.NewObject().Set("ok", true).Set("steps", []any{}), nil
}
func (r *cmdRecorder) Alive() bool { return true }
func (r *cmdRecorder) Close()      {}
func (r *cmdRecorder) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sent)
}

// connectedMaster is a Master wired to a fake worker with an open link (no real process).
func connectedMaster(process *cmdRecorder) *Master {
	m := NewMaster(&fakeProjects{}, nil, "", nil)
	m.mu.Lock()
	m.process, m.linkOpen = process, true
	m.mu.Unlock()
	return m
}

// TestCyclicStepsAddressAndTogglesValue: each shot adds the step to the address and flips an
// on/off value, so a single+toggle cycle walks 100/101/102 with values 1/0/1.
func TestCyclicStepsAddressAndTogglesValue(t *testing.T) {
	rec := &cmdRecorder{}
	m := connectedMaster(rec)
	command := pyjson.NewObject().Set("kind", "single").Set("address", int64(100)).Set("value", int64(1)).Set("select", false)
	if _, err := m.StartCyclic(command, 100, 1, true); err != nil {
		t.Fatal(err)
	}
	for end := time.Now().Add(2 * time.Second); rec.count() < 3 && time.Now().Before(end); time.Sleep(10 * time.Millisecond) {
	}
	m.StopCyclic()
	if rec.count() < 3 {
		t.Fatalf("only %d shots", rec.count())
	}
	rec.mu.Lock()
	shots := append([]*pyjson.Object(nil), rec.sent[:3]...)
	rec.mu.Unlock()
	wantAddress := []int64{100, 101, 102}
	wantValue := []int64{1, 0, 1}
	for i, shot := range shots {
		address, _ := pytext.Int64(get(shot, "address"))
		value, _ := pytext.Int64(get(shot, "value"))
		if address != wantAddress[i] || value != wantValue[i] {
			t.Fatalf("shot %d address %d value %d, want %d/%d", i, address, value, wantAddress[i], wantValue[i])
		}
	}
	if pytext.Truthy(get(m.CyclicState(), "running")) {
		t.Fatal("still running after Stop")
	}
}

// TestCyclicRejectsBadInterval: the cycle time must be 100-600000 ms and needs an open link.
func TestCyclicRejectsBadInterval(t *testing.T) {
	m := connectedMaster(&cmdRecorder{})
	command := pyjson.NewObject().Set("kind", "single").Set("address", int64(1)).Set("value", int64(1))
	if _, err := m.StartCyclic(command, 10, 0, false); err == nil {
		t.Fatal("expected a cycle-time error")
	}
	disconnected := NewMaster(&fakeProjects{}, nil, "", nil)
	if _, err := disconnected.StartCyclic(command, 1000, 0, false); err == nil {
		t.Fatal("expected a not-connected error")
	}
}

// TestToggledValue: on/off kinds flip, step/double flip 1<->2, others keep their value.
func TestToggledValue(t *testing.T) {
	cases := []struct {
		kind string
		base float64
		flip bool
		want int64
	}{{"single", 1, true, 0}, {"coil", 0, true, 1}, {"double", 2, true, 1}, {"regulating", 1, true, 2},
		{"single", 1, false, 1}, {"setpoint_float", 5, true, 5}}
	for _, c := range cases {
		got, _ := pytext.Int64(toggledValue(c.kind, c.base, c.flip))
		if got != c.want {
			t.Fatalf("toggledValue(%s,%v,%v)=%d want %d", c.kind, c.base, c.flip, got, c.want)
		}
	}
}
