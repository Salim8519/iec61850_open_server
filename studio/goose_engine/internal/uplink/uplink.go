// Package uplink owns the one Npcap handle that connects the engine to a real adapter.
// It is receive-only until the engine allows sending, so "Inside the app" can never
// transmit even by mistake.
package uplink

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"studio-goose-engine/internal/npcap"
)

// Port is what the engine needs from an uplink; tests provide a fake.
type Port interface {
	Send(frame []byte) bool
	AllowSend(allowed bool)
	OpenSince() (time.Time, bool)
	Close()
}

// opener opens the adapter; tests replace it with a fake device.
type opener func(guid string) (device, error)

// Uplink keeps the adapter open, reopening it every RetryEvery when it is lost.
type Uplink struct {
	guid     string
	open     opener
	sig      *signals
	out      chan []byte
	allow    atomic.Bool
	pending  atomic.Bool   // a send wake is already on its way
	wakes    atomic.Uint64 // returns from the wait, for the idle measurements
	onFrame  func([]byte)
	onState  func(state, message string)
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	openedAt time.Time
	isOpen   bool
	failing  bool
}

// RetryEvery is how often a missing or lost adapter is retried.
const RetryEvery = 2 * time.Second

// Start opens the adapter in the background. onState gets open, lost, error,
// needs_npcap and send_failed changes; onFrame gets every GOOSE frame heard.
func Start(guid string, onFrame func([]byte), onState func(state, message string)) *Uplink {
	return start(guid, openNpcap, onFrame, onState)
}

func start(guid string, open opener, onFrame func([]byte), onState func(state, message string)) *Uplink {
	u := &Uplink{guid: guid, open: open, out: make(chan []byte, 1024), onFrame: onFrame,
		onState: onState, stop: make(chan struct{}), done: make(chan struct{})}
	sig, err := newSignals()
	if err != nil {
		close(u.done)
		onState("error", "could not create the adapter wait events: "+err.Error())
		return u
	}
	u.sig = sig
	go u.run()
	return u
}

func openNpcap(guid string) (device, error) {
	handle, err := npcap.Open(guid)
	if err != nil {
		return nil, err
	}
	return handle, nil
}

// Send queues a frame and wakes the adapter thread at once. It refuses while
// sending is not allowed or the queue is full.
func (u *Uplink) Send(frame []byte) bool {
	if !u.allow.Load() || u.sig == nil {
		return false
	}
	select {
	case u.out <- frame:
		if u.pending.CompareAndSwap(false, true) {
			u.sig.kick()
		}
		return true
	default:
		return false
	}
}

// AllowSend switches transmission on or off.
func (u *Uplink) AllowSend(allowed bool) { u.allow.Store(allowed) }

// OpenSince reports when the adapter was opened, for the listen-before-share check.
func (u *Uplink) OpenSince() (time.Time, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.openedAt, u.isOpen
}

// Close stops the uplink and releases the adapter.
func (u *Uplink) Close() {
	u.allow.Store(false)
	close(u.stop)
	if u.sig != nil {
		u.sig.halt()
	}
	<-u.done
	if u.sig != nil {
		u.sig.release()
	}
}

func (u *Uplink) run() {
	runtime.LockOSThread()
	defer close(u.done)
	for {
		dev, err := u.open(u.guid)
		if err != nil {
			state := "error"
			if errors.Is(err, npcap.ErrMissing) {
				state = "needs_npcap"
			}
			u.onState(state, err.Error())
		} else {
			u.setOpen(true)
			u.onState("open", "")
			lost := u.pump(dev)
			dev.Close()
			u.setOpen(false)
			if lost == nil {
				return
			}
			u.onState("lost", lost.Error())
		}
		select {
		case <-u.stop:
			return
		case <-time.After(RetryEvery):
		}
	}
}

func (u *Uplink) setOpen(open bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.isOpen = open
	if open {
		u.openedAt = time.Now()
	}
}
