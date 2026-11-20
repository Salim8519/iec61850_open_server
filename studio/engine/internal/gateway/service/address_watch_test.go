package service

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// Change 0221, trial N8: before, nothing noticed that a running gateway's address was taken off
// its card; the page kept saying "Listening on". Now the status says "Address gone" within a few
// seconds and goes back when the address returns.
func TestAGoneListenAddressIsShownAndClears(t *testing.T) {
	previous := watchEvery
	watchEvery = 5 * time.Millisecond
	defer func() { watchEvery = previous }()
	hub := newHub()
	m := NewManager(&fakeProjects{section: gateway.EmptyGateway()}, hub, nil, "", nil)
	var present atomic.Bool
	present.Store(true)
	m.HasAddress = func(ip string) (bool, error) { return ip == "198.18.21.50" && present.Load(), nil }
	process := &fakeGatewayProcess{}
	m.mu.Lock()
	m.process, m.state, m.message = process, "running", "Listening on 198.18.21.50:2404"
	m.watch = m.watchAddress(process, "198.18.21.50", "198.18.21.50:2404")
	m.mu.Unlock()
	if m.watch == nil {
		t.Fatal("one address of this PC is watched")
	}
	time.Sleep(30 * time.Millisecond)
	if status := m.Status(); get(status, "address_gone") != nil || get(status, "message") != "Listening on 198.18.21.50:2404" {
		t.Fatalf("address there: %v", status)
	}
	present.Store(false) // taken off the card
	status := waitFor(t, m, true)
	message, _ := get(status, "message").(string)
	if !strings.HasPrefix(message, "Address gone: 198.18.21.50 is no longer on this PC") ||
		!strings.Contains(message, "Give the card its address back") || get(status, "state") != "running" {
		t.Fatalf("gone: %v", status)
	}
	present.Store(true) // back
	if status := waitFor(t, m, false); get(status, "message") != "Listening on 198.18.21.50:2404" {
		t.Fatalf("back: %v", status)
	}
	hub.mu.Lock()
	events := strings.Join(hub.events, ",")
	hub.mu.Unlock()
	if events != "gateway_address_gone,gateway_address_back" {
		t.Fatalf("events %v", hub.events)
	}
	if _, err := m.Stop(false); err != nil {
		t.Fatal(err)
	}
	present.Store(false) // after Stop nothing is watched any more
	time.Sleep(30 * time.Millisecond)
	if status := m.Status(); get(status, "address_gone") != nil || get(status, "state") != "stopped" {
		t.Fatalf("after stop: %v", status)
	}
}

func waitFor(t *testing.T, m *Manager, gone bool) *pyjson.Object {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if status := m.Status(); (get(status, "address_gone") == true) == gone {
			return status
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("address_gone never became %v", gone)
	return nil
}

// All networks and 127.0.0.1 cannot disappear; without a reader nothing is watched.
func TestOnlyOneAddressOfThisPCIsWatched(t *testing.T) {
	for bind, want := range map[string]bool{"0.0.0.0": false, "127.0.0.1": false, "": false, "198.18.21.50": true,
		"192.168.1.10": true} {
		if watched(bind) != want {
			t.Fatalf("watched(%q) = %v", bind, !want)
		}
	}
	m := NewManager(&fakeProjects{section: gateway.EmptyGateway()}, newHub(), nil, "", nil)
	if m.watchAddress(&fakeGatewayProcess{}, "198.18.21.50", "x") != nil {
		t.Fatal("watched without a reader")
	}
}
