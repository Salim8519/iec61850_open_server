package engine

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"studio-goose-engine/internal/uplink"
)

// fakePort stands in for an Npcap adapter: tests push "wire" frames and read what was sent.
type fakePort struct {
	allow   atomic.Bool
	onFrame func([]byte)
	onState func(string, string)
	mu      sync.Mutex
	sent    [][]byte
	opened  time.Time
}

func (f *fakePort) Send(b []byte) bool {
	if !f.allow.Load() {
		return false
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sent = append(f.sent, b)
	return true
}
func (f *fakePort) AllowSend(allowed bool)       { f.allow.Store(allowed) }
func (f *fakePort) OpenSince() (time.Time, bool) { return f.opened, !f.opened.IsZero() }
func (f *fakePort) Close()                       {}
func (f *fakePort) sentCount() int               { f.mu.Lock(); defer f.mu.Unlock(); return len(f.sent) }

type rig struct {
	t      *testing.T
	engine *Engine
	port   *fakePort
	events chan map[string]any
	inbox  *net.UDPConn
	worker *net.UDPConn
	busAt  *net.UDPAddr
}

func newRig(t *testing.T) *rig {
	t.Helper()
	listenBeforeShare = 50 * time.Millisecond
	r := &rig{t: t, events: make(chan map[string]any, 64), port: &fakePort{}}
	r.engine = New(func(event map[string]any) { r.events <- event },
		func(_ string, onFrame func([]byte), onState func(string, string)) uplink.Port {
			r.port.onFrame, r.port.onState = onFrame, onState
			return r.port
		})
	t.Cleanup(r.engine.Close)
	var err error
	if r.inbox, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}); err != nil {
		t.Fatal(err)
	}
	if r.worker, err = net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.inbox.Close(); r.worker.Close() })
	return r
}

func (r *rig) publish(b []byte) {
	if _, err := r.worker.WriteToUDP(b, r.busAt); err != nil {
		r.t.Fatal(err)
	}
}

// received returns the next inbox datagram, or nil after the timeout.
func (r *rig) received(timeout time.Duration) []byte {
	buffer := make([]byte, 4096)
	_ = r.inbox.SetReadDeadline(time.Now().Add(timeout))
	n, _, err := r.inbox.ReadFromUDP(buffer)
	if err != nil {
		return nil
	}
	return buffer[:n]
}

func (r *rig) event(kind string) map[string]any {
	r.t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case event := <-r.events:
			if event["event"] == kind {
				return event
			}
		case <-deadline:
			r.t.Fatalf("no %s event", kind)
		}
	}
}
