package frame_test

import (
	"testing"

	"studio-goose-engine/internal/frame"
	"studio-goose-engine/internal/frame/frametest"
)

var src = frame.MAC{0x02, 0x11, 0x22, 0x00, 0x00, 0x01}

func TestParseReadsTheRoutingFieldsOfAnUntaggedFrame(t *testing.T) {
	info, err := frame.Parse(frametest.Build(frametest.Options{Src: src, AppID: 0x3001, GocbRef: "IED1LD0/LLN0$GO$gcb1"}))
	if err != nil {
		t.Fatal(err)
	}
	if info.GocbRef != "IED1LD0/LLN0$GO$gcb1" || info.AppID != 0x3001 || info.Src != src || info.Tagged {
		t.Fatalf("unexpected %+v", info)
	}
	if info.Dst != frametest.GooseDst || !info.Dst.Multicast() {
		t.Fatalf("destination %v", info.Dst)
	}
}

func TestParseSkipsTheVlanTagAndLongLengths(t *testing.T) {
	info, err := frame.Parse(frametest.Build(frametest.Options{Src: src, GocbRef: "LD/LLN0$GO$x", Tagged: true, Padding: 300}))
	if err != nil {
		t.Fatal(err)
	}
	if !info.Tagged || info.GocbRef != "LD/LLN0$GO$x" {
		t.Fatalf("unexpected %+v", info)
	}
}

func TestParseRejectsOtherEtherTypesAndTruncatedFrames(t *testing.T) {
	good := frametest.Build(frametest.Options{Src: src, GocbRef: "LD/LLN0$GO$x"})
	sampled := append([]byte{}, good...)
	sampled[12], sampled[13] = 0x88, 0xBA
	for name, b := range map[string][]byte{"sv": sampled, "short": good[:20], "cut": good[:len(good)-3]} {
		if _, err := frame.Parse(b); err == nil {
			t.Errorf("%s frame was accepted", name)
		}
	}
}

func TestMacTextMatchesStudioLive(t *testing.T) {
	if got := src.String(); got != "02-11-22-00-00-01" {
		t.Fatal(got)
	}
}
