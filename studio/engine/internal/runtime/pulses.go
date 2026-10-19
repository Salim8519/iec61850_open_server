package runtime

import (
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// pulses are backend-owned Boolean pulses (simulation/pulses.py). Each pulse is a timer, not a
// 50 Hz loop; the previous value is restored with a revision guard on the IED's actor.
type pulses struct {
	c       *Coordinator
	mu      sync.Mutex
	order   []sourceKey
	pending map[sourceKey]*pulse
}

type pulse struct {
	id, iedID, ref string
	previous       any
	revision       any
	timer          *time.Timer
}

func newPulses(c *Coordinator) *pulses { return &pulses{c: c, pending: map[sourceKey]*pulse{}} }

// Pulse is POST /signals/pulse: {"id", "value"}.
func (c *Coordinator) Pulse(iedID, ref string, active, duration any) (*pyjson.Object, error) {
	return c.pulses.start(iedID, ref, active, duration)
}

func (p *pulses) start(iedID, ref string, active, duration any) (*pyjson.Object, error) {
	c := p.c
	flag, isBool := active.(bool)
	milliseconds, isInt := duration.(int64)
	if !isBool || !isInt || milliseconds < 100 || milliseconds > 60000 {
		return nil, apperr.ValueError("Pulse requires a Boolean value and 100 to 60000 milliseconds")
	}
	if !c.isLive() {
		return nil, apperr.ValueError("Run the IED before triggering a pulse")
	}
	target := c.simulatedRun(iedID)
	if target == nil {
		return nil, apperr.ValueError("Pulses can only be sent to simulated IEDs")
	}
	info, err := c.signalInfo(iedID, ref)
	if err != nil {
		return nil, err
	}
	if info.Kind != "BOOLEAN" {
		return nil, apperr.ValueError("Pulses require a Boolean signal")
	}
	var started *pulse
	err = target.do(func() error {
		before, err := target.worker.ReadVersioned(ref)
		if err != nil {
			return err
		}
		guard := pyval.Obj("ref", ref, "revision", pyval.Get(before, "revision"))
		_, guarded, err := c.writeSample(target, []item{{ref, flag, nil}}, "pulse", guard)
		if err != nil {
			return err
		}
		p.cancel(iedID, ref, true, "Replaced by a newer pulse")
		started = &pulse{id: newHexID(), iedID: iedID, ref: ref, previous: pyval.Get(before, "value"),
			revision: pyval.Get(guarded, "revision")}
		key := sourceKey{iedID, ref}
		p.mu.Lock()
		p.order = append(p.order, key)
		p.pending[key] = started
		started.timer = time.AfterFunc(time.Duration(milliseconds)*time.Millisecond, func() { p.complete(started) })
		p.mu.Unlock()
		c.emit("pulse_started", pyval.Obj("correlation_id", started.id, "ied_id", iedID, "ref", ref,
			"duration_ms", milliseconds))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return pyval.Obj("id", started.id, "value", flag), nil
}

// cancel removes matching pulses (iedID "" = all IEDs; refGiven false = every reference).
func (p *pulses) cancel(iedID, ref string, refGiven bool, reason string) {
	p.mu.Lock()
	var removed []*pulse
	kept := p.order[:0]
	for _, key := range p.order {
		if (iedID == "" || key.ied == iedID) && (!refGiven || key.ref == ref) {
			if found := p.pending[key]; found != nil {
				found.timer.Stop()
				removed = append(removed, found)
			}
			delete(p.pending, key)
			continue
		}
		kept = append(kept, key)
	}
	p.order = kept
	p.mu.Unlock()
	for _, found := range removed {
		p.c.emit("pulse_cancelled", pyval.Obj("correlation_id", found.id, "ied_id", found.iedID,
			"ref", found.ref, "message", reason))
	}
}

// complete restores the previous value when the pulse is still pending (Pulses.advance).
func (p *pulses) complete(due *pulse) {
	key := sourceKey{due.iedID, due.ref}
	p.mu.Lock()
	if p.pending[key] != due {
		p.mu.Unlock()
		return
	}
	delete(p.pending, key)
	for i, candidate := range p.order {
		if candidate == key {
			p.order = append(p.order[:i], p.order[i+1:]...)
			break
		}
	}
	p.mu.Unlock()
	c := p.c
	kind, message := "pulse_completed", "Previous value restored"
	target := c.simulatedRun(due.iedID)
	var err error
	if target == nil {
		err = errNotRunning
	} else {
		err = target.do(func() error {
			guard := pyval.Obj("ref", due.ref, "revision", due.revision)
			_, _, err := c.writeSample(target, []item{{due.ref, due.previous, nil}}, "pulse_return", guard)
			return err
		})
	}
	if err != nil {
		kind, message = "pulse_failed", errorText(err)
	}
	c.emit(kind, pyval.Obj("correlation_id", due.id, "ied_id", due.iedID, "ref", due.ref, "message", message))
}

func (p *pulses) close() { p.cancel("", "", false, "Simulation stopped") }

func (c *Coordinator) isLive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.live()
}
