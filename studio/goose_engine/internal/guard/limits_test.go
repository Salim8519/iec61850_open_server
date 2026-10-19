package guard_test

import (
	"testing"
	"time"

	"studio-goose-engine/internal/frame"
	"studio-goose-engine/internal/guard"
)

func TestRateCapRefusesExcessFramesAndResetsEachSecond(t *testing.T) {
	g := guard.New(3)
	g.Register("ied1", ours, []string{ref})
	frameInfo := info(t, ours, ref)
	for i := 0; i < 3; i++ {
		if ok, why := g.MaySend(frameInfo, now); !ok {
			t.Fatalf("frame %d refused: %s", i, why)
		}
	}
	if ok, why := g.MaySend(frameInfo, now.Add(500*time.Millisecond)); ok || why != "rate" {
		t.Fatalf("fourth frame in one second went out: %s", why)
	}
	if ok, _ := g.MaySend(frameInfo, now.Add(time.Second)); !ok {
		t.Fatal("a new second must allow frames again")
	}
}

func TestOutboundRefusesUnicastOversizedAndEmptyFrames(t *testing.T) {
	good := info(t, ours, ref)
	unicast := good
	unicast.Dst = frame.MAC{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	big := good
	big.Size = frame.MaxFrame + 1
	empty := good
	empty.GocbRef = ""
	if guard.ValidOutbound(good) != nil {
		t.Fatal("valid frame refused")
	}
	for name, candidate := range map[string]frame.Info{"unicast": unicast, "big": big, "empty": empty} {
		if guard.ValidOutbound(candidate) == nil {
			t.Errorf("%s frame accepted", name)
		}
	}
}
