package engine

import "studio-goose-engine/internal/frame"

// noteReal records GOOSE heard from a real device: a globally administered source MAC.
// Simulated IEDs of any ScadaBench use locally administered 02:… MACs, so they never count.
// The first real frame stops unconfirmed sharing before another simulated frame leaves.
func (e *Engine) noteReal(src frame.MAC) {
	if src[0]&0x02 != 0 {
		return
	}
	e.counts.realFrames.Add(1)
	if e.realSeen.Swap(true) {
		return
	}
	e.mu.Lock()
	held := e.realHeldLocked()
	if held {
		e.stopSharing()
	}
	e.mu.Unlock()
	if held {
		e.poke()
		e.emit(map[string]any{"event": "goose_share", "state": "real_hold"})
	}
}

// realHeldLocked reports whether sharing waits for the user because a real device is on
// the adapter and sharing next to it was not confirmed (ADR-0119). Call with mu held.
func (e *Engine) realHeldLocked() bool {
	return e.link.share && !e.link.confirmReal && e.realSeen.Load()
}
