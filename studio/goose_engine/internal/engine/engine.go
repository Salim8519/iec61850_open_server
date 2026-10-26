// Package engine joins the loopback bus, the smart guard and the optional uplink.
// Every routing decision for GOOSE in Studio is made here.
package engine

import (
	"crypto/rand"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"studio-goose-engine/internal/bus"
	"studio-goose-engine/internal/frame"
	"studio-goose-engine/internal/guard"
	"studio-goose-engine/internal/uplink"
)

// RatePerSecond caps the frames one IED may send outside per second.
const RatePerSecond = 1000

// PortFactory opens an uplink; tests replace it with a fake.
type PortFactory func(guid string, onFrame func([]byte), onState func(state, message string)) uplink.Port

// IED is one simulated publisher and the gocbRefs it owns.
type IED struct {
	ID       string   `json:"id"`
	GocbRefs []string `json:"gocb_refs"`
}

// Engine is created once per Run by the backend.
type Engine struct {
	emit     func(map[string]any)
	guard    *guard.Guard
	newPort  PortFactory
	bus      *bus.Bus
	counts   counters
	sharing  atomic.Bool
	realSeen atomic.Bool // GOOSE from a vendor (globally administered) MAC was heard on the adapter
	mu       sync.Mutex
	prefix   [2]byte
	next     uint32
	sources  map[string]frame.MAC
	link     linkState
	storms   map[string]time.Time
	stopTick chan struct{}
	wake     chan struct{} // the share check may have to start ticking
}

// New returns an engine that reports events through emit.
func New(emit func(map[string]any), newPort PortFactory) *Engine {
	e := &Engine{emit: emit, guard: guard.New(RatePerSecond), newPort: newPort,
		sources: map[string]frame.MAC{}, storms: map[string]time.Time{}, stopTick: make(chan struct{}),
		wake: make(chan struct{}, 1)}
	_, _ = rand.Read(e.prefix[:])
	go e.tick()
	return e
}

// Open starts the bus on first use and registers IEDs; calling it again adds IEDs
// imported during a Run. Each IED gets its own locally administered source MAC
// (02:<run-random 2 bytes>:<index>) so our frames are recognisable on any wire.
func (e *Engine) Open(ieds []IED) (map[string]any, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.bus == nil {
		b, err := bus.Listen()
		if err != nil {
			return nil, err
		}
		e.bus = b
		go b.Run(e.fromBus)
	}
	result := map[string]string{}
	for _, ied := range ieds {
		if ied.ID == "" {
			return nil, errors.New("every IED needs an id")
		}
		mac, known := e.sources[ied.ID]
		if !known {
			e.next++
			mac = frame.MAC{0x02, e.prefix[0], e.prefix[1], byte(e.next >> 16), byte(e.next >> 8), byte(e.next)}
			e.sources[ied.ID] = mac
		}
		e.guard.Register(ied.ID, mac, ied.GocbRefs)
		result[ied.ID] = mac.String()
	}
	return map[string]any{"port": e.bus.Port(), "sources": result}, nil
}

// Deliver points accepted frames at studio_live's inbox port.
func (e *Engine) Deliver(port int) error {
	e.mu.Lock()
	b := e.bus
	e.mu.Unlock()
	if b == nil {
		return errors.New("open the engine before setting the inbox")
	}
	return b.SetInbox(port)
}

// Gate applies the user's answer to a conflict.
func (e *Engine) Gate(ied, choice string) (guard.Gate, error) { return e.guard.Choose(ied, choice) }

// fromBus handles a frame published by a simulated IED.
func (e *Engine) fromBus(b []byte) {
	e.counts.busFrames.Add(1)
	info, err := frame.Parse(b)
	if err != nil || !e.guard.FromApp(info) {
		e.counts.busRejected.Add(1)
		return
	}
	if e.bus.Deliver(bus.OriginApp, b) {
		e.counts.deliveredApp.Add(1)
	} else {
		e.counts.noInbox.Add(1)
	}
	if !e.sharing.Load() {
		return
	}
	ok, why := e.guard.MaySend(info, time.Now())
	if !ok {
		e.counts.refuse(why)
		if why == "rate" {
			e.storm(info)
		}
		return
	}
	e.mu.Lock()
	port := e.link.port
	e.mu.Unlock()
	if port != nil && port.Send(b) {
		e.counts.sent.Add(1)
	} else {
		e.counts.refuse("queue")
	}
}

// fromWire handles a frame heard on the adapter.
func (e *Engine) fromWire(b []byte) {
	e.counts.wireFrames.Add(1)
	info, err := frame.Parse(b)
	if err != nil {
		return
	}
	e.noteReal(info.Src)
	deliver, why, conflict := e.guard.FromWire(info)
	if conflict != nil {
		e.emit(map[string]any{"event": "goose_conflict", "ied_id": conflict.IED, "gocb_ref": conflict.GocbRef,
			"src_mac": conflict.Src, "sharing": e.sharing.Load()})
	}
	if !deliver {
		e.counts.refuse(why)
		return
	}
	if e.bus.Deliver(bus.OriginWire, b) {
		e.counts.deliveredWire.Add(1)
	}
}

func (e *Engine) storm(info frame.Info) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for ied, mac := range e.sources {
		if mac == info.Src && time.Since(e.storms[ied]) > 5*time.Second {
			e.storms[ied] = time.Now()
			e.emit(map[string]any{"event": "goose_storm", "ied_id": ied, "limit_per_second": RatePerSecond})
		}
	}
}

// Close stops sharing, the uplink and the bus.
func (e *Engine) Close() {
	close(e.stopTick)
	e.SetUplink("", false, false)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.bus != nil {
		e.bus.Close()
		e.bus = nil
	}
}
