package frame_test

import (
	"testing"

	"studio-goose-engine/internal/frame"
	"studio-goose-engine/internal/frame/frametest"
)

// The 26-byte VLAN-tagged GOOSE header that once read past the end of the slice.
var taggedHeaderOnly = []byte{
	0x01, 0x0C, 0xCD, 0x01, 0x00, 0x01, 0x02, 0x11, 0x22, 0x00, 0x00, 0x01,
	0x81, 0x00, 0x80, 0x00, 0x88, 0xB8, 0x10, 0x00, 0x00, 0x08, 0, 0, 0, 0,
}

func TestParseRejectsTheTaggedHeaderWithoutAPdu(t *testing.T) {
	if _, err := frame.Parse(taggedHeaderOnly); err == nil {
		t.Fatal("a frame without goosePdu was accepted")
	}
}

func TestParseRejectsEveryTruncationOfAValidFrame(t *testing.T) {
	cases := map[string][]byte{
		"untagged":      frametest.Build(frametest.Options{Src: src, GocbRef: "LD/LLN0$GO$x"}),
		"tagged":        frametest.Build(frametest.Options{Src: src, GocbRef: "LD/LLN0$GO$x", Tagged: true}),
		"tagged-long":   frametest.Build(frametest.Options{Src: src, GocbRef: "LD/LLN0$GO$x", Tagged: true, Padding: 300}),
		"untagged-long": frametest.Build(frametest.Options{Src: src, GocbRef: "LD/LLN0$GO$x", Padding: 300}),
	}
	for name, good := range cases {
		if _, err := frame.Parse(good); err != nil {
			t.Fatalf("%s: the full frame was refused: %v", name, err)
		}
		// Every cut shorter than the end of the gocbRef must be refused without a panic.
		refEnd := len(good) - 300
		if name == "untagged" || name == "tagged" {
			refEnd = len(good)
		}
		for cut := 0; cut < len(good); cut++ {
			info, err := parseSafely(t, good[:cut])
			if cut < refEnd && err == nil {
				t.Errorf("%s cut at %d was accepted: %+v", name, cut, info)
			}
		}
	}
}

func parseSafely(t *testing.T, b []byte) (info frame.Info, err error) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Parse panicked on %d bytes: %v", len(b), r)
		}
	}()
	return frame.Parse(b)
}

func FuzzParse(f *testing.F) {
	f.Add(frametest.Build(frametest.Options{Src: src, GocbRef: "IED1LD0/LLN0$GO$gcb1"}))
	f.Add(frametest.Build(frametest.Options{Src: src, GocbRef: "LD/LLN0$GO$x", Tagged: true, Padding: 300}))
	f.Add(taggedHeaderOnly)
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		info, err := frame.Parse(b)
		if err == nil && (info.Size != len(b) || len(info.GocbRef) > len(b)) {
			t.Fatalf("inconsistent result %+v for %d bytes", info, len(b))
		}
	})
}
