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

// Uplink keeps the adapter open, reopening it every RetryEvery when it is lost.
type Uplink struct {
	guid     string
	out      chan []byte
	allow    atomic.Bool
	onFrame  func([]byte)
	onState  func(state, message string)
	stop     chan struct{}
	done     chan struct{}
	mu       sync.Mutex
	openedAt time.Time
	open     bool
	failing  bool
}

// RetryEvery is how often a missing or lost adapter is retried.
const RetryEvery = 2 * time.Second

// Start opens the adapter in the background. onState gets open, lost, error,
// needs_npcap and send_failed changes; onFrame gets every GOOSE frame heard.
func Start(guid string, onFrame func([]byte), onState func(state, message string)) *Uplink {
	u := &Uplink{guid: guid, out: make(chan []byte, 1024), onFrame: onFrame, onState: onState,
		stop: make(chan struct{}), done: make(chan struct{})}
	go u.run()
	return u
}

// Send queues a frame. It refuses while sending is not allowed or the queue is full.
func (u *Uplink) Send(frame []byte) bool {
	if !u.allow.Load() {
		return false
	}
	select {
	case u.out <- frame:
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
	return u.openedAt, u.open
}

// Close stops the uplink and releases the adapter.
func (u *Uplink) Close() {
	u.allow.Store(false)
	close(u.stop)
	<-u.done
}

func (u *Uplink) run() {
	runtime.LockOSThread()
	defer close(u.done)
	for {
		handle, err := npcap.Open(u.guid)
		if err != nil {
			state := "error"
			if errors.Is(err, npcap.ErrMissing) {
				state = "needs_npcap"
			}
			u.onState(state, err.Error())
		} else {
			u.setOpen(true)
			u.onState("open", "")
			lost := u.pump(handle)
			handle.Close()
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

// pump alternates a 1 ms receive wait with draining the outbound queue, so the
// handle is only ever used from this goroutine.
func (u *Uplink) pump(handle *npcap.Handle) error {
	for {
		select {
		case <-u.stop:
			return nil
		default:
		}
		frame, err := handle.Next()
		if err != nil {
			return err
		}
		if frame != nil {
			u.onFrame(frame)
		}
		u.flush(handle)
	}
}

func (u *Uplink) flush(handle *npcap.Handle) {
	for i := 0; i < 256; i++ {
		select {
		case frame := <-u.out:
			if !u.allow.Load() {
				continue
			}
			if err := handle.Send(frame); err != nil {
				if !u.failing {
					u.failing = true
					u.onState("send_failed", err.Error())
				}
			} else if u.failing {
				u.failing = false
				u.onState("open", "")
			}
		default:
			return
		}
	}
}

func (u *Uplink) setOpen(open bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.open = open
	if open {
		u.openedAt = time.Now()
	}
}
