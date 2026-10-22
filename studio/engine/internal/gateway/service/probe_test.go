package service

import (
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// stoppableProcess is studio_master's probe: it runs until probe_stop arrives (change 0178).
type stoppableProcess struct {
	stop chan struct{}
	ops  chan string
}

func (s *stoppableProcess) Request(op string, _ time.Duration, _ *pyjson.Object) (any, error) {
	s.ops <- op
	if op == "probe_stop" {
		close(s.stop)
		return true, nil
	}
	<-s.stop
	found := []any{pyjson.NewObject().Set("table", "holding_registers").Set("start", int64(0)).Set("count", int64(2))}
	return pyjson.NewObject().Set("found", found).Set("requests", int64(7)).Set("stopped", true), nil
}
func (s *stoppableProcess) Alive() bool { return true }
func (s *stoppableProcess) Close()      {}

// TestProbeStop (0178, C14): Stop sends probe_stop to the worker, the probe ends "stopped"
// with the ranges found so far, and Stop with nothing running sends nothing.
func TestProbeStop(t *testing.T) {
	process := &stoppableProcess{stop: make(chan struct{}), ops: make(chan string, 4)}
	probe := &Probe{}
	ended := make(chan *pyjson.Object, 1)
	if _, err := probe.Start(process, "holding_registers", 0, 9999, func(snapshot *pyjson.Object) { ended <- snapshot }); err != nil {
		t.Fatal(err)
	}
	if op := <-process.ops; op != "probe" {
		t.Fatalf("first op %s", op)
	}
	if _, err := probe.Stop(); err != nil {
		t.Fatal(err)
	}
	snapshot := <-ended
	text, _ := pyjson.Compact(snapshot)
	want := `{"state":"stopped","table":"holding_registers","start":0,"end":9999,"requests":7,"address":0,` +
		`"found":[{"table":"holding_registers","start":0,"count":2}],"message":"Stopped by you after 7 requests."}`
	if string(text) != want {
		t.Fatalf("snapshot %s", text)
	}
	if _, err := probe.Stop(); err != nil || len(process.ops) != 1 { // only the first probe_stop
		t.Fatalf("stop when idle: %v, ops %d", err, len(process.ops))
	}
}
