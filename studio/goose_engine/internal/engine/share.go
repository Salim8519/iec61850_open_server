package engine

import (
	"time"

	"studio-goose-engine/internal/uplink"
)

// listenBeforeShare is how long the adapter must have been listening before any
// GOOSE goes out, so a real device already sending our gocbRef is caught first.
var listenBeforeShare = 3 * time.Second

type linkState struct {
	port    uplink.Port
	adapter string
	share   bool
	state   string
	message string
}

// SetUplink listens on the adapter ("" closes it) and, when share is true, sends
// simulated GOOSE outside after the listen window. It returns the new status.
func (e *Engine) SetUplink(adapter string, share bool) map[string]any {
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
		if adapter != "" {
			e.link.state = "opening"
			e.link.port = e.newPort(adapter, e.fromWire, func(state, message string) { e.uplinkState(adapter, state, message) })
		}
	}
	wasShare := e.link.share
	e.link.share = share && adapter != ""
	nowShare := e.link.share
	if !nowShare {
		e.stopSharing()
	}
	status := e.statusLocked()
	e.mu.Unlock()
	if nowShare && !wasShare {
		e.emit(map[string]any{"event": "goose_share", "state": "checking"})
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
	e.emit(map[string]any{"event": "goose_uplink", "adapter_guid": adapter, "state": state, "message": message})
}

// tick opens sharing once the listen window has passed with the adapter open.
func (e *Engine) tick() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-e.stopTick:
			return
		case <-ticker.C:
		}
		e.mu.Lock()
		ready := false
		if e.link.share && e.link.port != nil && !e.sharing.Load() && e.link.state == "open" {
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
		if e.sharing.Load() {
			share = "sharing"
		}
	}
	return map[string]any{"adapter_guid": e.link.adapter, "uplink": e.link.state, "message": e.link.message,
		"share": share}
}
