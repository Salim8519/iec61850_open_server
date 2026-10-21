package push

import (
	"testing"
	"time"

	"studio-engine/internal/events"
)

func TestListenSeesBusEvents(t *testing.T) {
	bus := events.NewBus(events.Capacity)
	seen := make(chan string, 4)
	area := New(Options{Bus: bus, Listen: func(event *events.Event) { seen <- event.Kind }})
	defer area.Close()
	time.Sleep(10 * time.Millisecond) // the reader starts after the events published so far
	bus.Publish("signal", nil)
	bus.Publish("project_changed", nil)
	select {
	case kind := <-seen:
		if kind != "project_changed" {
			t.Fatalf("signal events are not passed on: %s", kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Listen not called")
	}
}
