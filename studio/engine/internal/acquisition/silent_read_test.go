package acquisition

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// silentDevice is a live client whose device stopped answering while the connection stays
// open (a frozen relay, a pulled cable): studio_live answers every read, but each point
// carries libiec61850's IED_ERROR_TIMEOUT (20) instead of a value.
type silentDevice struct{ codes []int64 }

func (d silentDevice) Request(op string, _ time.Duration, fields *pyjson.Object) (any, error) {
	if op != "read" {
		return pyjson.NewObject(), nil
	}
	var values []any
	for index, raw := range pyv.GetList(fields, "refs") {
		entry := pyv.O("ref", pyv.Get(pyv.Obj(raw), "ref"))
		if code := d.codes[index%len(d.codes)]; code >= 0 {
			entry.Set("error", code)
		} else {
			entry.Set("value", 1.0)
		}
		values = append(values, entry)
	}
	return pyv.O("values", values), nil
}
func (silentDevice) Close() {}

func pollingSession(t *testing.T, device silentDevice) (*RealSession, *testValues, string) {
	t.Helper()
	ref := "LD0/MMXU1.TotW.mag.f"
	ied := pyv.O("id", "real1", "name", "Bay1", "role", "real",
		"live", pyv.O("ip", "127.0.0.9", "port", int64(102), "access", "poll", "poll_ms", int64(10)),
		"signals", []any{pyv.O("ref", ref, "value", 1.0)})
	project := pyv.O("ieds", []any{ied})
	values, events := newTestValues(project), &testEvents{}
	deps := Deps{Projects: testProjects{project}, Events: events, Bus: events, Values: values, States: testIdle{}, Equipment: testIdle{},
		Clock: testClock{start: time.Now()}}
	values.SetLive("real1", []RefValue{{Ref: ref, Value: 2.0}})
	session := NewRealSession(deps, NewLiveValues(deps), ied, nil)
	session.put("polling", []any{pyv.O("ref", ref, "fc", "MX"), pyv.O("ref", "LD0/MMXU1.Hz.mag.f", "fc", "MX")})
	session.set("live", "Receiving live values by polling")
	session.mu.Lock()
	session.process = device
	session.resetLostLocked()
	session.mu.Unlock()
	return session, values, ref
}

// A device that answers no point of a poll (every point timed out) ends the live loop and its
// values are marked stale, like a lost connection (fix plan A18, found live on Windows: a frozen
// IED kept "Receiving live values" with frozen values shown as fresh).
func TestSilentDeviceMarksValuesStale(t *testing.T) {
	session, values, ref := pollingSession(t, silentDevice{codes: []int64{20}})
	done := make(chan error, 1)
	go func() { done <- session.live() }()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "did not answer") {
			t.Fatalf("live() ended with %v, want the device-did-not-answer error", err)
		}
	case <-time.After(2 * time.Second):
		session.stopOnce.Do(func() { close(session.stopping) })
		t.Fatal("the live loop kept running while the device answered nothing")
	}
	if _, quality, _ := values.Get("real1", ref); quality != "stale" {
		t.Fatalf("value is %v after the device stopped answering, want stale", quality)
	}
}

// A device that answers some points is alive: a point it refuses (object does not exist, 22)
// or one timeout among answers does not end the session.
func TestPartlyAnsweringDeviceStaysLive(t *testing.T) {
	for _, codes := range [][]int64{{22}, {-1, 20}} {
		session, _, _ := pollingSession(t, silentDevice{codes: codes})
		done := make(chan error, 1)
		go func() { done <- session.live() }()
		select {
		case err := <-done:
			t.Fatalf("codes %v: live() ended with %v", codes, err)
		case <-time.After(300 * time.Millisecond):
		}
		session.stopOnce.Do(func() { close(session.stopping) })
		<-done
	}
}

// The mode refresh (every 10 s while reports carry the values) also notices a silent device.
func TestSilentModeReadIsAnError(t *testing.T) {
	ied := pyv.O("id", "real1", "name", "Bay1", "signals", []any{pyv.O("ref", "LD0/LLN0.Mod.stVal", "fc", "ST")})
	if _, err := readMode(silentDevice{codes: []int64{20, 3}}, ied, time.Now); err == nil ||
		!strings.Contains(err.Error(), "did not answer") {
		t.Fatalf("readMode of a silent device: %v", err)
	}
	if _, err := readMode(silentDevice{codes: []int64{22}}, ied, time.Now); err != nil {
		t.Fatalf("a refused mode point is an answer: %v", err)
	}
}
