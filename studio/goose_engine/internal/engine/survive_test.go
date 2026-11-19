package engine

import (
	"bytes"
	"testing"
	"time"

	"studio-goose-engine/internal/frame/frametest"
	"studio-goose-engine/internal/uplink"
)

// panicPort fails like a broken adapter driver: every Send panics.
type panicPort struct{ *fakePort }

func (panicPort) Send([]byte) bool { panic("adapter send exploded") }

func refusedPanics(e *Engine) uint64 { return e.Stats()["refused"].(map[string]uint64)["panic"] }

func TestAPanickingBusFrameIsCountedAndTheBusKeepsRunning(t *testing.T) {
	r := newRig(t)
	r.engine.newPort = func(_ string, onFrame func([]byte), onState func(string, string)) uplink.Port {
		r.port.onFrame, r.port.onState = onFrame, onState
		return panicPort{r.port}
	}
	mac := r.opened()
	r.sharing()
	sent := frametest.Build(frametest.Options{Src: mac, GocbRef: ref})
	r.publish(sent)
	r.received(time.Second)
	deadline := time.Now().Add(time.Second)
	for refusedPanics(r.engine) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if refusedPanics(r.engine) != 1 {
		t.Fatalf("panic counted %d times", refusedPanics(r.engine))
	}
	r.publish(sent)
	if got := r.received(time.Second); got == nil || !bytes.Equal(got[1:], sent) {
		t.Fatalf("bus stopped after the panic: %x", got)
	}
}

func TestAPanickingWireFrameIsCountedAndDoesNotEscape(t *testing.T) {
	e := New(func(map[string]any) { panic("event writer exploded") }, nil)
	defer e.Close()
	e.guard.Register("ied1", frametest.GooseDst, []string{ref})
	conflicting := frametest.Build(frametest.Options{Src: device, GocbRef: ref})
	e.fromWire(conflicting) // the conflict event panics inside the handler
	if refusedPanics(e) != 1 {
		t.Fatalf("panic counted %d times", refusedPanics(e))
	}
}

func TestFramesAfterCloseAreDroppedWithoutTouchingTheBus(t *testing.T) {
	e := New(func(map[string]any) {}, nil)
	result, err := e.Open([]IED{{ID: "ied1", GocbRefs: []string{ref}}})
	if err != nil {
		t.Fatal(err)
	}
	e.Close()
	if mac, ok := e.sources["ied1"]; !ok || result["port"] == nil {
		t.Fatal("ied1 was not opened")
	} else {
		e.fromBus(frametest.Build(frametest.Options{Src: mac, GocbRef: ref}))
	}
	e.fromWire(frametest.Build(frametest.Options{Src: device, GocbRef: "RELAY/LLN0$GO$trip"}))
	if refusedPanics(e) != 0 || e.Stats()["no_inbox"] != uint64(1) {
		t.Fatalf("frames after Close: %v", e.Stats())
	}
}
