package push

import (
	"sync"
	"time"
)

// throttle coalesces "something changed" marks into runs of fire: the first mark after a
// quiet gap fires at once (leading edge); marks during a run or within gap of the last
// start fire once more, gap after that start (trailing edge). It never blocks the caller
// and arms no timer while nothing is marked.
type throttle struct {
	clock Clock
	gap   time.Duration
	fire  func() // must not block; the hub queues the build

	mu      sync.Mutex
	last    time.Time // start of the last run (zero before the first)
	timer   Timer     // armed trailing run
	running bool      // fired and not yet done
	again   bool      // marked while running
	closed  bool
}

func newThrottle(clock Clock, gap time.Duration, fire func()) *throttle {
	return &throttle{clock: clock, gap: gap, fire: fire}
}

// mark asks for a run.
func (t *throttle) mark() {
	t.mu.Lock()
	if t.closed || t.timer != nil {
		t.mu.Unlock()
		return
	}
	if t.running {
		t.again = true
		t.mu.Unlock()
		return
	}
	now := t.clock.Now()
	if wait := t.last.Add(t.gap).Sub(now); !t.last.IsZero() && wait > 0 {
		t.timer = t.clock.AfterFunc(wait, t.expired)
		t.mu.Unlock()
		return
	}
	t.running, t.last = true, now
	t.mu.Unlock()
	t.fire()
}

func (t *throttle) expired() {
	t.mu.Lock()
	if t.closed || t.timer == nil {
		t.mu.Unlock()
		return
	}
	t.timer, t.running, t.last = nil, true, t.clock.Now()
	t.mu.Unlock()
	t.fire()
}

// done ends a run; a mark that arrived meanwhile schedules the trailing run.
func (t *throttle) done() {
	t.mu.Lock()
	t.running = false
	again := t.again
	t.again = false
	t.mu.Unlock()
	if again {
		t.mark()
	}
}

// close stops a pending run for good.
func (t *throttle) close() {
	t.mu.Lock()
	t.closed = true
	if t.timer != nil {
		t.timer.Stop()
		t.timer = nil
	}
	t.mu.Unlock()
}

// pending reports an armed timer (tests and idle checks).
func (t *throttle) pending() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.timer != nil
}
