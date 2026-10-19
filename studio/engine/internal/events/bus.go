// Package events is the application event bus and its SSE stream, identical to the Python
// engine's runtime/event_bus.py and api/events.py: sequence ids from 1 per process, the
// last 5000 events kept for replay, a synthetic "resync" when a cursor fell out of the
// ring, every event encoded once with json.dumps rules and shared by all clients.
package events

import (
	"sync"
	"time"

	"studio-engine/internal/pycompat"
	"studio-engine/internal/pyjson"
)

// Capacity is the number of events kept for replay (EventBus(capacity=5000)).
const Capacity = 5000

// Event is one published event. It is immutable: Payload and Data are shared by every
// subscriber and must never be modified.
type Event struct {
	ID        int64
	Kind      string
	Timestamp float64
	// Payload holds the keys after the envelope, in call order.
	Payload *pyjson.Object
	// Data is json.dumps(event): {"id": .., "kind": .., "timestamp": .., payload...}.
	Data []byte
}

// Bus is a bounded, sequenced event ring with reconnect cursors.
type Bus struct {
	mu       sync.Mutex
	ring     []*Event
	start    int
	count    int
	sequence int64
	changed  chan struct{}
	now      func() float64
}

// NewBus returns an empty bus keeping capacity events.
func NewBus(capacity int) *Bus {
	return &Bus{ring: make([]*Event, capacity), changed: make(chan struct{}), now: wallClock}
}

// SetClock replaces time.time() (tests).
func (b *Bus) SetClock(now func() float64) { b.now = now }

func wallClock() float64 { return pycompat.TimeTime(time.Now()) }

// Publish appends {"id", "kind", "timestamp", **payload} and wakes every waiting reader.
// payload may be nil; it is kept as the event's Payload and must not change afterwards.
// The payload is encoded before the lock; only the envelope is written under it.
func (b *Bus) Publish(kind string, payload *pyjson.Object) (*Event, error) {
	if payload == nil {
		payload = pyjson.NewObject()
	}
	fragment, plain, err := encodePayload(payload)
	if err != nil {
		return nil, err
	}
	b.mu.Lock()
	b.sequence++
	event := &Event{ID: b.sequence, Kind: kind, Timestamp: b.now(), Payload: payload}
	if plain {
		event.Data = envelope(event.ID, kind, event.Timestamp, fragment)
	} else if event.Data, err = encodeOverridden(event); err != nil {
		b.sequence--
		b.mu.Unlock()
		return nil, err
	}
	b.store(event)
	close(b.changed)
	b.changed = make(chan struct{})
	b.mu.Unlock()
	return event, nil
}

func (b *Bus) store(event *Event) {
	if b.count < len(b.ring) {
		b.ring[(b.start+b.count)%len(b.ring)] = event
		b.count++
		return
	}
	b.ring[b.start] = event
	b.start = (b.start + 1) % len(b.ring)
}

// After returns the events with an id above cursor, as EventBus.after: with wait > 0 and
// nothing newer it waits once for the next publish, the timeout or done. A non-zero cursor
// older than the ring gives one unstored "resync" event with the current sequence.
func (b *Bus) After(cursor int64, wait time.Duration, done <-chan struct{}) []*Event {
	b.mu.Lock()
	if wait > 0 && b.sequence <= cursor {
		changed := b.changed
		b.mu.Unlock()
		timer := time.NewTimer(wait)
		select {
		case <-changed:
		case <-timer.C:
		case <-done:
		}
		timer.Stop()
		b.mu.Lock()
	}
	defer b.mu.Unlock()
	if b.count == 0 {
		return nil
	}
	oldest := b.ring[b.start].ID
	if cursor != 0 && cursor < oldest-1 {
		return []*Event{b.resync()}
	}
	first := cursor + 1
	if first < oldest {
		first = oldest
	}
	var items []*Event
	for id := first; id <= b.sequence; id++ {
		items = append(items, b.ring[(b.start+int(id-oldest))%len(b.ring)])
	}
	return items
}

func (b *Bus) resync() *Event {
	event := &Event{ID: b.sequence, Kind: "resync", Timestamp: b.now(),
		Payload: pyjson.NewObject()}
	event.Data = envelope(event.ID, event.Kind, event.Timestamp, nil)
	return event
}

// Sequence returns the id of the last published event (0 before the first).
func (b *Bus) Sequence() int64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.sequence
}
