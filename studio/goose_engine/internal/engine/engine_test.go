package engine

import (
	"bytes"
	"net"
	"testing"
	"time"

	"studio-goose-engine/internal/frame"
	"studio-goose-engine/internal/frame/frametest"
)

const ref = "IED1LD0/LLN0$GO$gcb1"

var device = frame.MAC{0x00, 0x30, 0xA7, 0x10, 0x20, 0x30}

// opened registers ied1, points the inbox at the rig and returns ied1's MAC.
func (r *rig) opened() frame.MAC {
	r.t.Helper()
	result, err := r.engine.Open([]IED{{ID: "ied1", GocbRefs: []string{ref}}})
	if err != nil {
		r.t.Fatal(err)
	}
	r.busAt = &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: result["port"].(int)}
	if err := r.engine.Deliver(r.inbox.LocalAddr().(*net.UDPAddr).Port); err != nil {
		r.t.Fatal(err)
	}
	text := result["sources"].(map[string]string)["ied1"]
	var mac frame.MAC
	for i := range mac {
		var b byte
		for _, c := range text[i*3 : i*3+2] {
			b = b<<4 | byte(map[rune]int{'0': 0, '1': 1, '2': 2, '3': 3, '4': 4, '5': 5, '6': 6, '7': 7, '8': 8, '9': 9,
				'A': 10, 'B': 11, 'C': 12, 'D': 13, 'E': 14, 'F': 15}[c])
		}
		mac[i] = b
	}
	if mac[0] != 0x02 {
		r.t.Fatalf("source MAC %s is not locally administered", text)
	}
	return mac
}

func (r *rig) sharing() {
	r.t.Helper()
	r.engine.SetUplink("{GUID}", true, true)
	r.port.opened = time.Now().Add(-time.Second)
	r.port.onState("open", "")
	r.event("goose_share") // checking
	if state := r.event("goose_share")["state"]; state != "sharing" {
		r.t.Fatalf("share state %v", state)
	}
}

func TestSimulatedGooseReachesTheInboxWithoutAnyAdapter(t *testing.T) {
	r := newRig(t)
	mac := r.opened()
	sent := frametest.Build(frametest.Options{Src: mac, GocbRef: ref})
	r.publish(sent)
	got := r.received(time.Second)
	if got == nil || got[0] != 0 || !bytes.Equal(got[1:], sent) {
		t.Fatalf("inbox got %x", got)
	}
}

func TestFramesFromUnregisteredSendersNeverReachTheInbox(t *testing.T) {
	r := newRig(t)
	r.opened()
	r.publish(frametest.Build(frametest.Options{Src: device, GocbRef: ref}))
	if got := r.received(200 * time.Millisecond); got != nil {
		t.Fatalf("stranger delivered: %x", got)
	}
}

func TestInsideModeListensButNeverTransmits(t *testing.T) {
	r := newRig(t)
	mac := r.opened()
	r.engine.SetUplink("{GUID}", false, true)
	r.port.opened = time.Now().Add(-time.Minute)
	r.port.onState("open", "")
	time.Sleep(200 * time.Millisecond)
	r.publish(frametest.Build(frametest.Options{Src: mac, GocbRef: ref}))
	r.received(time.Second)
	heard := frametest.Build(frametest.Options{Src: device, GocbRef: "RELAY/LLN0$GO$trip"})
	r.port.onFrame(heard)
	got := r.received(time.Second)
	if got == nil || got[0] != 1 || !bytes.Equal(got[1:], heard) {
		t.Fatalf("real GOOSE not delivered as wire: %x", got)
	}
	if r.port.sentCount() != 0 || r.port.allow.Load() {
		t.Fatal("inside mode transmitted")
	}
}

func TestSharingStartsAfterTheListenWindowAndSendsOwnFrames(t *testing.T) {
	r := newRig(t)
	mac := r.opened()
	r.sharing()
	r.publish(frametest.Build(frametest.Options{Src: mac, GocbRef: ref}))
	r.received(time.Second)
	deadline := time.Now().Add(time.Second)
	for r.port.sentCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if r.port.sentCount() != 1 {
		t.Fatalf("sent %d frames", r.port.sentCount())
	}
}

func TestConflictHoldsTheIedAndContinueReleasesIt(t *testing.T) {
	r := newRig(t)
	mac := r.opened()
	r.sharing()
	r.port.onFrame(frametest.Build(frametest.Options{Src: device, GocbRef: ref}))
	conflict := r.event("goose_conflict")
	if conflict["ied_id"] != "ied1" || conflict["src_mac"] != device.String() || conflict["sharing"] != true {
		t.Fatalf("conflict %v", conflict)
	}
	if got := r.received(200 * time.Millisecond); got != nil {
		t.Fatal("the real device's copy of our gocbRef reached the app")
	}
	own := frametest.Build(frametest.Options{Src: mac, GocbRef: ref})
	r.publish(own)
	r.received(time.Second)
	time.Sleep(50 * time.Millisecond)
	if r.port.sentCount() != 0 {
		t.Fatal("held IED was sent outside")
	}
	if _, err := r.engine.Gate("ied1", "continue"); err != nil {
		t.Fatal(err)
	}
	r.publish(own)
	r.received(time.Second)
	time.Sleep(50 * time.Millisecond)
	if r.port.sentCount() != 1 {
		t.Fatalf("continued IED sent %d frames", r.port.sentCount())
	}
}

func TestOwnReflectionFromTheWireIsDropped(t *testing.T) {
	r := newRig(t)
	mac := r.opened()
	r.sharing()
	r.port.onFrame(frametest.Build(frametest.Options{Src: mac, GocbRef: ref}))
	if got := r.received(200 * time.Millisecond); got != nil {
		t.Fatal("reflection delivered")
	}
	if r.engine.Stats()["refused"].(map[string]uint64)["reflection"] != 1 {
		t.Fatal("reflection not counted")
	}
}

func TestConflictSeenWhileListeningIsAskedAgainWhenSharingStarts(t *testing.T) {
	r := newRig(t)
	r.opened()
	r.engine.SetUplink("{GUID}", false, true)
	r.port.onFrame(frametest.Build(frametest.Options{Src: device, GocbRef: ref}))
	if r.event("goose_conflict")["sharing"] != false {
		t.Fatal("listen-only conflict should say sharing is off")
	}
	r.engine.SetUplink("{GUID}", true, true)
	if r.event("goose_conflict")["sharing"] != true {
		t.Fatal("waiting conflict not raised again when sharing starts")
	}
}
