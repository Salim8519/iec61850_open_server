package engine

import (
	"time"

	"studio-goose-engine/internal/uplink"
)

// listenBeforeShare is how long the adapter must have been listening before any
// GOOSE goes out, so a real device already sending our gocbRef is caught first.
var listenBeforeShare = 3 * time.Second

type linkState struct {
	port        uplink.Port
	adapter     string
	share       bool
	confirmReal bool // the user confirmed sharing next to real devices on this adapter
	state       string
	message     string
}

// SetUplink listens on the adapter ("" closes it) and, when share is true, sends
// simulated GOOSE outside after the listen window. Without confirmReal, sharing holds
// as soon as a real device is heard (ADR-0119). It returns the new status.
func (e *Engine) SetUplink(adapter string, share, confirmReal bool) map[string]any {
	e.mu.Lock()
	if adapter != e.link.adapter && e.link.port != nil {
		old := e.link.port
		e.link.port = nil
		e.sharing.Store(false)
		e.mu.Unlock()
		old.Close() // waits for its goroutine; never hold the lock here
		e.mu.Lock()
	}
	if adapter != e.link.adapter {
		e.link = linkState{adapter: adapter, state: "closed"}
		e.realSeen.Store(false)
		if adapter != "" {
			e.link.state = "opening"
			e.link.port = e.newPort(adapter, e.fromWire, func(state, message string) { e.uplinkState(adapter, state, message) })
		}
	}
	wasShare := e.link.share
	e.link.share = share && adapter != ""
	e.link.confirmReal = confirmReal
	nowShare := e.link.share
	if e.realHeldLocked() {
		e.stopSharing()
	}
	if !nowShare {
		e.stopSharing()
	}
	status := e.statusLocked()
	e.mu.Unlock()
	e.poke()
	if nowShare && !wasShare {
		e.emit(map[string]any{"event": "goose_share", "state": status["share"]})
		for _, conflict := range e.guard.Waiting() {
			e.emit(map[string]any{"event": "goose_conflict", "ied_id": conflict.IED, "gocb_ref": conflict.GocbRef,
				"src_mac": conflict.Src, "sharing": true})
		}
	}
	if wasShare && !nowShare {
		e.emit(map[string]any{"event": "goose_share", "state": "off"})
	}
	return status
}

func (e *Engine) stopSharing() {
	e.sharing.Store(false)
	if e.link.port != nil {
		e.link.port.AllowSend(false)
	}
}

func (e *Engine) uplinkState(adapter, state, message string) {
	e.mu.Lock()
	if e.link.adapter != adapter || (e.link.state == state && e.link.message == message) {
		e.mu.Unlock()
		return
	}
	e.link.state, e.link.message = state, message
	if state != "open" {
		e.stopSharing()
	}
	e.mu.Unlock()
	e.poke()
	e.emit(map[string]any{"event": "goose_uplink", "adapter_guid": adapter, "state": state, "message": message})
}

// tickEvery is how often the listen window is checked while sharing is waiting for it.
const tickEvery = 100 * time.Millisecond

// tick opens sharing once the listen window has passed with the adapter open. It
// ticks only while that is pending, so an idle engine never wakes.
func (e *Engine) tick() {
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()
	ticking := true
	for {
		if pending := e.sharePending(); pending != ticking {
			ticking = pending
			if ticking {
				ticker.Reset(tickEvery)
			} else {
				ticker.Stop()
			}
		}
		select {
		case <-e.stopTick:
			return
		case <-e.wake:
			continue
		case <-ticker.C:
		}
		e.mu.Lock()
		ready := false
		if e.link.share && e.link.port != nil && !e.sharing.Load() && e.link.state == "open" && !e.realHeldLocked() {
			if since, open := e.link.port.OpenSince(); open && time.Since(since) >= listenBeforeShare {
				e.link.port.AllowSend(true)
				e.sharing.Store(true)
				ready = true
			}
		}
		e.mu.Unlock()
		if ready {
			e.emit(map[string]any{"event": "goose_share", "state": "sharing"})
		}
	}
}

// sharePending reports whether sharing waits for the listen window on an open adapter.
func (e *Engine) sharePending() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.link.share && e.link.port != nil && !e.sharing.Load() && e.link.state == "open" && !e.realHeldLocked()
}

// poke makes tick re-check whether it has to tick.
func (e *Engine) poke() {
	select {
	case e.wake <- struct{}{}:
	default:
	}
}

// Status describes the uplink for the backend.
func (e *Engine) Status() map[string]any {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.statusLocked()
}

func (e *Engine) statusLocked() map[string]any {
	share := "off"
	if e.link.share {
		share = "checking"
		if e.realHeldLocked() {
			share = "real_hold"
		} else if e.sharing.Load() {
			share = "sharing"
		}
	}
	return map[string]any{"adapter_guid": e.link.adapter, "uplink": e.link.state, "message": e.link.message,
		"share": share}
}
