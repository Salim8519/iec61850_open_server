package engine

import (
	"sync"
	"sync/atomic"
)

type counters struct {
	busFrames     atomic.Uint64
	busRejected   atomic.Uint64
	deliveredApp  atomic.Uint64
	noInbox       atomic.Uint64
	wireFrames    atomic.Uint64
	deliveredWire atomic.Uint64
	realFrames    atomic.Uint64
	sent          atomic.Uint64
	mu            sync.Mutex
	refused       map[string]uint64
}

func (c *counters) refuse(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.refused == nil {
		c.refused = map[string]uint64{}
	}
	c.refused[reason]++
}

// Stats is the engine snapshot for the GOOSE traffic view and the evidence logs.
func (e *Engine) Stats() map[string]any {
	e.counts.mu.Lock()
	refused := make(map[string]uint64, len(e.counts.refused))
	for reason, count := range e.counts.refused {
		refused[reason] = count
	}
	e.counts.mu.Unlock()
	e.mu.Lock()
	sources := make(map[string]string, len(e.sources))
	for ied, mac := range e.sources {
		sources[ied] = mac.String()
	}
	e.mu.Unlock()
	return map[string]any{
		"bus_frames": e.counts.busFrames.Load(), "bus_rejected": e.counts.busRejected.Load(),
		"delivered_app": e.counts.deliveredApp.Load(), "no_inbox": e.counts.noInbox.Load(),
		"wire_frames": e.counts.wireFrames.Load(), "delivered_wire": e.counts.deliveredWire.Load(),
		"real_frames": e.counts.realFrames.Load(),
		"sent":        e.counts.sent.Load(), "refused": refused, "gates": e.guard.Gates(),
		"sources": sources, "link": e.Status(),
	}
}
