package runtime

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// Changes in a row give one warm-up after the quiet delay, and Run ends with its context.
func TestStartWarmerWaitsForTheLastChange(t *testing.T) {
	var reads atomic.Int32
	warmer := NewStartWarmer(func() *pyjson.Object {
		reads.Add(1)
		return pyjson.NewObject().Set("ieds", []any{pyjson.NewObject().Set("id", "a")})
	}, 40*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	ended := make(chan struct{})
	go func() { warmer.Run(ctx); close(ended) }()
	for range 5 {
		warmer.Changed()
		time.Sleep(10 * time.Millisecond)
	}
	if reads.Load() != 0 {
		t.Fatal("warmed while the project was still changing")
	}
	deadline := time.Now().Add(2 * time.Second)
	for reads.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(100 * time.Millisecond)
	if got := reads.Load(); got != 1 {
		t.Fatalf("project read %d times; want one warm-up", got)
	}
	cancel()
	select {
	case <-ended:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not end with its context")
	}
}

// Changed never blocks, even when nobody runs the warmer (tests without background loops).
func TestStartWarmerChangedNeverBlocks(t *testing.T) {
	warmer := NewStartWarmer(func() *pyjson.Object { return nil }, time.Hour)
	done := make(chan struct{})
	go func() {
		for range 100 {
			warmer.Changed()
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Changed blocked")
	}
}

// An IED read once is skipped while its text stays; a new or changed text is read again.
func TestStartWarmerSkipsIedsItAlreadyRead(t *testing.T) {
	warmer := NewStartWarmer(nil, 0)
	first := pyjson.NewObject().Set("id", "a").Set("name", "A").Set("source", "<SCL/>")
	warmer.warm(context.Background(), pyjson.NewObject().Set("ieds", []any{first}))
	if warmer.warmed[warmKey{"a", "A"}] != "<SCL/>" {
		t.Fatal("first warm-up did not record the IED")
	}
	moved := pyjson.NewObject().Set("id", "a").Set("name", "A").Set("source", "<SCL/>")
	added := pyjson.NewObject().Set("id", "b").Set("name", "B").Set("source", "<SCL></SCL>")
	warmer.warm(context.Background(), pyjson.NewObject().Set("ieds", []any{moved, added}))
	if len(warmer.warmed) != 2 || warmer.warmed[warmKey{"b", "B"}] != "<SCL></SCL>" {
		t.Fatalf("second warm-up recorded %v; want both IEDs", warmer.warmed)
	}
	warmer.warm(context.Background(), pyjson.NewObject().Set("ieds", []any{added}))
	if _, held := warmer.warmed[warmKey{"a", "A"}]; held {
		t.Fatal("a removed IED's text is still held")
	}
}
