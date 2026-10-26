package labnet

import (
	"bytes"
	"testing"
)

// The hardware ID list once came back empty (a NUL in a Go string) and crashed Create (change 0192).
func TestHardwareIDIsOneMultiString(t *testing.T) {
	want := []byte{'*', 0, 'M', 0, 'S', 0, 'L', 0, 'O', 0, 'O', 0, 'P', 0, 0, 0, 0, 0}
	if got := multiString(loopbackHardwareID); !bytes.Equal(got, want) {
		t.Fatalf("got % x", got)
	}
}
