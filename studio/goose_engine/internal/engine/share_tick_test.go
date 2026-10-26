package engine

import (
	"testing"
	"time"
)

// The listen-window check only ticks while sharing waits for it, so every way into
// that wait must wake it: share switched on, the adapter opening, the adapter coming back.
func TestSharingResumesWhenALostAdapterComesBack(t *testing.T) {
	r := newRig(t)
	r.opened()
	r.sharing()
	r.port.onState("lost", "cable pulled")
	if state := r.event("goose_uplink")["state"]; state != "lost" {
		t.Fatalf("uplink state %v", state)
	}
	if r.engine.sharing.Load() {
		t.Fatal("still sharing after the adapter was lost")
	}
	r.port.opened = time.Now()
	r.port.onState("open", "")
	if state := r.event("goose_share")["state"]; state != "sharing" {
		t.Fatalf("share state %v", state)
	}
}

func TestSharingWaitsForAnAdapterThatOpensLate(t *testing.T) {
	r := newRig(t)
	r.opened()
	r.engine.SetUplink("{GUID}", true, true)
	r.event("goose_share") // checking
	r.port.onState("needs_npcap", "Npcap is not installed")
	time.Sleep(3 * tickEvery)
	if r.engine.sharing.Load() || r.engine.sharePending() {
		t.Fatal("sharing started without an open adapter")
	}
	r.port.opened = time.Now()
	r.port.onState("open", "")
	if state := r.event("goose_share")["state"]; state != "sharing" {
		t.Fatalf("share state %v", state)
	}
}
