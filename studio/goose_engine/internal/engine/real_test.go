package engine

import (
	"testing"
	"time"

	"studio-goose-engine/internal/frame"
	"studio-goose-engine/internal/frame/frametest"
)

// ADR-0119: a real device on the adapter holds unconfirmed sharing before any frame
// leaves; GOOSE from another ScadaBench (02:… MAC) never does; confirming resumes it.
func TestARealDeviceHoldsUnconfirmedSharingUntilConfirmed(t *testing.T) {
	r := newRig(t)
	mac := r.opened()
	r.engine.SetUplink("{GUID}", true, false)
	r.event("goose_share") // checking
	r.port.onFrame(frametest.Build(frametest.Options{Src: device, GocbRef: "REAL/LLN0$GO$x"}))
	if state := r.event("goose_share")["state"]; state != "real_hold" {
		t.Fatalf("share state %v", state)
	}
	r.port.opened = time.Now().Add(-time.Second)
	r.port.onState("open", "")
	time.Sleep(3 * tickEvery)
	if r.engine.sharing.Load() || r.engine.Status()["share"] != "real_hold" {
		t.Fatalf("shared next to an unconfirmed real device: %v", r.engine.Status())
	}
	r.publish(frametest.Build(frametest.Options{Src: mac, GocbRef: ref}))
	r.received(time.Second)
	if r.port.sentCount() != 0 {
		t.Fatal("a frame left while held")
	}
	if r.engine.Stats()["real_frames"] != uint64(1) {
		t.Fatalf("real frames %v", r.engine.Stats()["real_frames"])
	}
	r.engine.SetUplink("{GUID}", true, true)
	if state := r.event("goose_share")["state"]; state != "sharing" {
		t.Fatalf("share state after confirm %v", state)
	}
}

func TestARealDeviceStopsSharingThatAlreadyRuns(t *testing.T) {
	r := newRig(t)
	r.opened()
	r.engine.SetUplink("{GUID}", true, false)
	r.port.opened = time.Now().Add(-time.Second)
	r.port.onState("open", "")
	r.event("goose_share") // checking
	if state := r.event("goose_share")["state"]; state != "sharing" {
		t.Fatalf("share state %v", state)
	}
	r.port.onFrame(frametest.Build(frametest.Options{Src: device, GocbRef: "REAL/LLN0$GO$x"}))
	if state := r.event("goose_share")["state"]; state != "real_hold" {
		t.Fatalf("share state %v", state)
	}
	if r.engine.sharing.Load() || r.port.allow.Load() {
		t.Fatal("still sending after a real device appeared")
	}
}

func TestAnotherScadaBenchIsNotARealDevice(t *testing.T) {
	r := newRig(t)
	r.opened()
	other := frame.MAC{0x02, 0x99, 0x98, 0, 0, 1}
	r.engine.SetUplink("{GUID}", true, false)
	r.port.onFrame(frametest.Build(frametest.Options{Src: other, GocbRef: "PEER/LLN0$GO$x"}))
	r.port.opened = time.Now().Add(-time.Second)
	r.port.onState("open", "")
	r.event("goose_share") // checking
	if state := r.event("goose_share")["state"]; state != "sharing" {
		t.Fatalf("share state %v", state)
	}
}
