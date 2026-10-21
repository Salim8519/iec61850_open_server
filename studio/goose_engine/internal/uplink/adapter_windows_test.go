package uplink

import (
	"os"
	"testing"
)

// TestRealAdapterIdleWakes listens on a real Npcap adapter, receive-only, and reports
// how often the adapter thread woke. It runs only when STUDIO_GOOSE_ADAPTER names an
// adapter GUID ({...}, or Loopback); sending is never allowed, so no frame goes out.
func TestRealAdapterIdleWakes(t *testing.T) {
	guid := os.Getenv("STUDIO_GOOSE_ADAPTER")
	if guid == "" {
		t.Skip("set STUDIO_GOOSE_ADAPTER to an adapter GUID to listen on a real adapter")
	}
	rec := newRecorder()
	u := Start(guid, rec.frame, rec.state)
	defer u.Close()
	rec.await(t, "open")
	seconds := idleSeconds()
	wakes := measureIdle(u, seconds)
	frames := rec.frameCount()
	t.Logf("adapter %s: %.1f wakes/s over %d s, %d GOOSE frames heard, states %v",
		guid, wakes, seconds, frames, rec.states)
	if frames == 0 && wakes > 0 {
		t.Fatalf("adapter thread woke %.1f times/s with no GOOSE traffic", wakes)
	}
}
