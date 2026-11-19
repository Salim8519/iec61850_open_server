package acquisition

import (
	"errors"
	"testing"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// failingRead is a live client whose poll read fails (the device stopped answering).
type failingRead struct{}

func (failingRead) Request(op string, _ time.Duration, _ *pyjson.Object) (any, error) {
	if op == "read" {
		return nil, errors.New("Native read request timed out")
	}
	return pyjson.NewObject(), nil
}
func (failingRead) Close() {}

// A poll read error ends the live loop and marks the IED's values stale, as a lost
// connection does (fix plan A18).
func TestReadErrorMarksValuesStale(t *testing.T) {
	ref := "LD0/MMXU1.TotW.mag.f"
	ied := pyv.O("id", "real1", "name", "Bay1", "role", "real",
		"live", pyv.O("ip", "127.0.0.9", "port", int64(102), "access", "poll", "poll_ms", int64(10)),
		"signals", []any{pyv.O("ref", ref, "value", 1.0)})
	project := pyv.O("ieds", []any{ied})
	values, events := newTestValues(project), &testEvents{}
	deps := Deps{Events: events, Bus: events, Values: values, States: testIdle{}, Clock: testClock{start: time.Now()}}
	values.SetLive("real1", []RefValue{{Ref: ref, Value: 2.0}})
	session := NewRealSession(deps, NewLiveValues(deps), ied, nil)
	session.put("polling", []any{pyv.O("ref", ref, "fc", "MX")})
	session.mu.Lock()
	session.process = failingRead{}
	session.resetLostLocked()
	session.mu.Unlock()
	if err := session.live(); err == nil {
		t.Fatal("the read error ends the live loop")
	}
	if _, quality, _ := values.Get("real1", ref); quality != "stale" {
		t.Fatalf("value is %v after a read error, want stale", quality)
	}
}
