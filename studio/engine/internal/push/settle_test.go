package push

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/events"
	"studio-engine/internal/pyjson"
)

// writes records every Write of the stream separately (each is one flush to the browser).
type writes struct {
	mu     sync.Mutex
	header http.Header
	parts  []string
}

func (w *writes) Header() http.Header { return w.header }
func (w *writes) WriteHeader(int)     {}
func (w *writes) Flush()              {}
func (w *writes) Write(p []byte) (int, error) {
	w.mu.Lock()
	w.parts = append(w.parts, string(p))
	w.mu.Unlock()
	return len(p), nil
}

func (w *writes) find(marker string) (string, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, part := range w.parts {
		if strings.Contains(part, marker) {
			return part, true
		}
	}
	return "", false
}

// A request publishes its events microseconds apart (a test press: signal, then
// equipment_state). After a quiet period they must leave in one write, also when an earlier
// burst was written before: without the settle the first left alone and the symbol's event
// waited out the whole gather window.
func TestFirstBurstAfterQuietLeavesTogether(t *testing.T) {
	bus := events.NewBus(events.Capacity)
	area := New(Options{Bus: bus, Epoch: "boot-1", Heartbeat: time.Hour, Gather: 200 * time.Millisecond,
		Settle: 40 * time.Millisecond})
	t.Cleanup(area.Close)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/push?client=tab-1", nil).WithContext(ctx)
	out := &writes{header: http.Header{}}
	done := make(chan struct{})
	go func() { defer close(done); _ = area.stream(out, request) }()
	deadline := time.Now().Add(3 * time.Second)
	for _, ok := out.find(`"t":"hello"`); !ok; _, ok = out.find(`"t":"hello"`) {
		if time.Now().After(deadline) {
			t.Fatal("no hello frame")
		}
		time.Sleep(time.Millisecond)
	}
	// An earlier burst, written and gathered: the stream then waits quietly on the bus.
	earlier := publish(t, bus, "activity", pyjson.NewObject().Set("message", "earlier"))
	for _, ok := out.find("id: " + itoa(earlier) + "\n"); !ok; _, ok = out.find("id: " + itoa(earlier) + "\n") {
		if time.Now().After(deadline) {
			t.Fatal("the earlier event never left")
		}
		time.Sleep(time.Millisecond)
	}
	time.Sleep(300 * time.Millisecond) // longer than Gather: the next press starts a new burst
	first := publish(t, bus, "signal", pyjson.NewObject().Set("ref", "LD0/XCBR1.Pos.stVal").Set("value", int64(2)))
	time.Sleep(5 * time.Millisecond)
	second := publish(t, bus, "equipment_state", pyjson.NewObject().Set("node_id", "CB1").Set("value", int64(2)))
	started := time.Now()
	for {
		if part, ok := out.find("id: " + itoa(first) + "\n"); ok {
			if !strings.Contains(part, "id: "+itoa(second)+"\n") {
				t.Fatalf("the burst left in two writes; first write: %q", part)
			}
			break
		}
		if time.Since(started) > 3*time.Second {
			t.Fatal("the events never left")
		}
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
}
