package push

import (
	"sort"
	"sync"
	"time"
)

// fakeClock moves only when the test advances it; due timers run in the advancing goroutine.
type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

type fakeTimer struct {
	clock *fakeClock
	at    time.Time
	f     func()
	done  bool
}

func newFakeClock() *fakeClock { return &fakeClock{now: time.Unix(1_000_000, 0)} }

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	timer := &fakeTimer{clock: c, at: c.now.Add(d), f: f}
	c.timers = append(c.timers, timer)
	return timer
}

func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	stopped := !t.done
	t.done = true
	return stopped
}

// armed counts timers that have not run or been stopped.
func (c *fakeClock) armed() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	count := 0
	for _, timer := range c.timers {
		if !timer.done {
			count++
		}
	}
	return count
}

// advance moves time forward by d, running every timer that falls due, in time order.
func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	end := c.now.Add(d)
	c.mu.Unlock()
	for {
		c.mu.Lock()
		sort.SliceStable(c.timers, func(i, j int) bool { return c.timers[i].at.Before(c.timers[j].at) })
		var due *fakeTimer
		for _, timer := range c.timers {
			if !timer.done && !timer.at.After(end) {
				due = timer
				break
			}
		}
		if due == nil {
			c.now = end
			c.mu.Unlock()
			return
		}
		due.done = true
		c.now = due.at
		c.mu.Unlock()
		due.f()
	}
}
