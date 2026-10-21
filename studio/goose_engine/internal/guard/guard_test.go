package guard_test

import (
	"testing"
	"time"

	"studio-goose-engine/internal/frame"
	"studio-goose-engine/internal/frame/frametest"
	"studio-goose-engine/internal/guard"
)

var (
	ours   = frame.MAC{0x02, 0xAA, 0xBB, 0x00, 0x00, 0x01}
	device = frame.MAC{0x00, 0x30, 0xA7, 0x10, 0x20, 0x30}
	now    = time.Unix(1000, 0)
)

const ref = "IED1LD0/LLN0$GO$gcb1"

func info(t *testing.T, src frame.MAC, gocbRef string) frame.Info {
	t.Helper()
	parsed, err := frame.Parse(frametest.Build(frametest.Options{Src: src, GocbRef: gocbRef}))
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func registered() *guard.Guard {
	g := guard.New(1000)
	g.Register("ied1", ours, []string{ref})
	return g
}

func TestAppFramesNeedARegisteredSourceAndOwnGocbRef(t *testing.T) {
	g := registered()
	if !g.FromApp(info(t, ours, ref)) || g.FromApp(info(t, device, ref)) {
		t.Fatal("app delivery must follow the registered MAC")
	}
	if ok, why := g.MaySend(info(t, ours, ref), now); !ok {
		t.Fatalf("own frame refused: %s", why)
	}
	if ok, why := g.MaySend(info(t, ours, "OTHER/LLN0$GO$x"), now); ok || why != "foreign_ref" {
		t.Fatalf("foreign gocbRef went out: %s", why)
	}
	if ok, why := g.MaySend(info(t, device, ref), now); ok || why != "unknown_source" {
		t.Fatalf("unknown source went out: %s", why)
	}
}

func TestOurReflectionIsDroppedAndNeverAConflict(t *testing.T) {
	g := registered()
	deliver, why, conflict := g.FromWire(info(t, ours, ref))
	if deliver || why != "reflection" || conflict != nil || g.Gates()["ied1"] != guard.Open {
		t.Fatalf("reflection handled as %v %s %v", deliver, why, conflict)
	}
}

func TestRealDeviceWithOurGocbRefHoldsTheGateOnceAndIsNeverDelivered(t *testing.T) {
	g := registered()
	deliver, _, conflict := g.FromWire(info(t, device, ref))
	if deliver || conflict == nil || conflict.IED != "ied1" || conflict.Src != device.String() {
		t.Fatalf("conflict not raised: %v %v", deliver, conflict)
	}
	again, _, repeat := g.FromWire(info(t, device, ref))
	if again || repeat != nil {
		t.Fatal("the same conflict must not be raised twice or delivered")
	}
	if ok, why := g.MaySend(info(t, ours, ref), now); ok || why != "held" {
		t.Fatalf("held gate let a frame out: %s", why)
	}
	if waiting := g.Waiting(); len(waiting) != 1 || waiting[0].GocbRef != ref {
		t.Fatalf("waiting %v", waiting)
	}
}

func TestStopBlocksAndContinueForcesTheOutwardCopy(t *testing.T) {
	g := registered()
	g.FromWire(info(t, device, ref))
	if gate, _ := g.Choose("ied1", "stop"); gate != guard.Blocked {
		t.Fatal(gate)
	}
	if ok, why := g.MaySend(info(t, ours, ref), now); ok || why != "blocked" {
		t.Fatalf("blocked gate let a frame out: %s", why)
	}
	if gate, _ := g.Choose("ied1", "continue"); gate != guard.Forced {
		t.Fatal(gate)
	}
	if ok, _ := g.MaySend(info(t, ours, ref), now); !ok {
		t.Fatal("forced gate refused")
	}
	if deliver, _, conflict := g.FromWire(info(t, device, ref)); deliver || conflict != nil {
		t.Fatal("a forced IED's wire copy must still stay outside the app")
	}
	if len(g.Waiting()) != 0 {
		t.Fatal("answered conflicts are not waiting")
	}
}

func TestChooseRejectsUnknownIedsAndChoices(t *testing.T) {
	g := registered()
	if _, err := g.Choose("nobody", "stop"); err == nil {
		t.Fatal("unknown IED accepted")
	}
	if _, err := g.Choose("ied1", "maybe"); err == nil {
		t.Fatal("unknown choice accepted")
	}
}

func TestRealDevicesWithOtherGocbRefsAreDelivered(t *testing.T) {
	deliver, why, conflict := registered().FromWire(info(t, device, "RELAY/LLN0$GO$trip"))
	if !deliver || why != "" || conflict != nil {
		t.Fatalf("unrelated device GOOSE refused: %s", why)
	}
}
