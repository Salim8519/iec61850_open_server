package runtime

import (
	"math"
	"time"

	"studio-engine/internal/runtime/pyval"
)

// cadence is simulation/cadence.py: fixed 0.1 s deadlines, at most 5 ticks of backlog, and the
// same float arithmetic for lateness and skipped ticks.
type cadence struct {
	period     float64
	maxBacklog int64
	deadline   float64
}

func newCadence(now float64) *cadence {
	c := &cadence{period: 0.1, maxBacklog: 5}
	c.reset(now)
	return c
}

func (c *cadence) reset(now float64) { c.deadline = now + c.period }

// take returns (lateness, skipped, true) when a tick is due.
func (c *cadence) take(now float64) (float64, int64, bool) {
	if now < c.deadline {
		return 0, 0, false
	}
	lateness := now - c.deadline
	due := int64(math.Floor((lateness+float64(c.period*1e-8))/c.period)) + 1
	skipped := due - c.maxBacklog
	if skipped < 0 {
		skipped = 0
	}
	c.deadline += float64(float64(skipped+1) * c.period)
	return lateness, skipped, true
}

// tick is Coordinator._tick: wait for the deadline, then run scenarios and automation.
func (c *Coordinator) tick() {
	defer c.ticker.Done()
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		c.mu.Lock()
		wait := c.cadence.deadline - c.now()
		c.mu.Unlock()
		if wait < 0 {
			wait = 0
		}
		timer.Reset(time.Duration(wait * float64(time.Second)))
		select {
		case <-c.shutdown:
			return
		case <-timer.C:
		}
		c.cycle()
	}
}

func (c *Coordinator) cycle() {
	c.lifeMu.Lock()
	defer c.lifeMu.Unlock()
	c.mu.Lock()
	now := c.now()
	if c.state != "running" {
		c.cadence.reset(now)
		c.lastAutomation = now
		c.mu.Unlock()
		return
	}
	lateness, skipped, due := c.cadence.take(now)
	if !due {
		c.mu.Unlock()
		return
	}
	delta := now - c.lastAutomation
	c.lastAutomation = now
	c.addElapsed(delta)
	elapsed := c.elapsed
	late := lateness >= c.cadence.period
	c.mu.Unlock()
	if late {
		c.emit("automation_late", pyval.Obj("lateness_seconds", lateness, "skipped_ticks", skipped))
	}
	if err := c.advance(elapsed); err != nil {
		c.mu.Lock()
		c.state = "paused"
		c.mu.Unlock()
		c.emit("automation_failed", pyval.Obj("message", errorText(err)))
		c.emit("runtime", pyval.Obj("state", "paused", "message", "Automation failed; MMS remains active"))
	}
}

func (c *Coordinator) advance(elapsed float64) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = panicError(recovered)
		}
	}()
	c.scenarios.advance(elapsed)
	c.automate(elapsed)
	return nil
}
