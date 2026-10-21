package xsd

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestFrameRoundTrip(t *testing.T) {
	var stream bytes.Buffer
	sent := request{Version: 1, ID: "7", Op: "validate", SchemaPath: `C:\é\SCL.xsd`, SourcePath: "a<b>&c"}
	if err := writeFrame(&stream, sent); err != nil {
		t.Fatal(err)
	}
	if err := writeFrame(&stream, map[string]any{"id": "8", "ok": true}); err != nil {
		t.Fatal(err)
	}
	size := binary.LittleEndian.Uint32(stream.Bytes()[:4])
	if encoded, _ := json.Marshal(sent); int(size) != len(encoded) {
		t.Fatalf("length header %d does not match the first payload", size)
	}
	payload, err := readFrame(&stream)
	if err != nil {
		t.Fatal(err)
	}
	var received request
	if err := json.Unmarshal(payload, &received); err != nil || received != sent {
		t.Fatalf("round trip changed the request: %+v %v", received, err)
	}
	if payload, err = readFrame(&stream); err != nil || string(payload) != `{"id":"8","ok":true}` {
		t.Fatalf("second frame: %q %v", payload, err)
	}
	if _, err := readFrame(&stream); err != io.EOF {
		t.Fatalf("want io.EOF at a clean end, got %v", err)
	}
}

func TestFrameRejectsBadLengths(t *testing.T) {
	for name, data := range map[string][]byte{
		"zero":      {0, 0, 0, 0},
		"too large": {1, 0, 0, 1},
		"header":    {5, 0},
		"payload":   {5, 0, 0, 0, '{', '}'},
	} {
		if _, err := readFrame(bytes.NewReader(data)); err == nil || err == io.EOF {
			t.Errorf("%s: want an error, got %v", name, err)
		}
	}
	huge := strings.Repeat("x", frameLimit)
	if err := writeFrame(io.Discard, huge); err == nil {
		t.Fatal("a frame above 16 MiB must be refused")
	}
}
