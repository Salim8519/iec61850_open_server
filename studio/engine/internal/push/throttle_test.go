package push

import (
	"testing"
	"time"
)

// runs records each fire as the fake time it happened at, completing it at once.
type runs struct {
	clock *fakeClock
	t     *throttle
	at    []time.Duration
	start time.Time
	hold  bool // leave the run open (a slow build)
}

func newRuns(gap time.Duration) *runs {
	r := &runs{clock: newFakeClock()}
	r.start = r.clock.Now()
	r.t = newThrottle(r.clock, gap, func() {
		r.at = append(r.at, r.clock.Now().Sub(r.start))
		if !r.hold {
			r.t.done()
		}
	})
	return r
}

func TestThrottleLeadingEdgeThenTrailing(t *testing.T) {
	r := newRuns(100 * time.Millisecond)
	r.t.mark() // quiet: at once
	if len(r.at) != 1 || r.at[0] != 0 {
		t.Fatalf("leading edge: %v", r.at)
	}
	for step := 0; step < 5; step++ { // a burst within the gap: one trailing run
		r.clock.advance(10 * time.Millisecond)
		r.t.mark()
	}
	if len(r.at) != 1 || !r.t.pending() {
		t.Fatalf("burst must wait for the gap: %v", r.at)
	}
	r.clock.advance(100 * time.Millisecond)
	if len(r.at) != 2 || r.at[1] != 100*time.Millisecond {
		t.Fatalf("trailing run at +100 ms: %v", r.at)
	}
	if r.t.pending() || r.clock.armed() != 0 {
		t.Fatal("no timer may stay armed after the trailing run")
	}
	r.clock.advance(time.Second) // idle: nothing runs, nothing is armed
	r.t.mark()
	if len(r.at) != 3 || r.at[2] != 1150*time.Millisecond {
		t.Fatalf("leading edge after quiet: %v", r.at)
	}
}

func TestThrottleSteadyMarksRunOncePerGap(t *testing.T) {
	r := newRuns(100 * time.Millisecond)
	for step := 0; step < 100; step++ { // a mark every 5 ms for 500 ms
		r.t.mark()
		r.clock.advance(5 * time.Millisecond)
	}
	r.clock.advance(time.Second)
	if len(r.at) != 6 {
		t.Fatalf("500 ms of marks at a 100 ms gap must run 6 times, got %d: %v", len(r.at), r.at)
	}
	for index := 1; index < len(r.at); index++ {
		if gap := r.at[index] - r.at[index-1]; gap < 100*time.Millisecond {
			t.Fatalf("runs %v closer than the gap", r.at)
		}
	}
}

func TestThrottleMarkDuringRunSchedulesOneMore(t *testing.T) {
	r := newRuns(250 * time.Millisecond)
	r.hold = true
	r.t.mark()
	r.t.mark()
	r.t.mark()
	if len(r.at) != 1 {
		t.Fatalf("a running build is not started twice: %v", r.at)
	}
	r.clock.advance(400 * time.Millisecond) // the build takes 400 ms
	r.hold = false
	r.t.done()
	if len(r.at) != 2 || r.at[1] != 400*time.Millisecond {
		t.Fatalf("a mark during a long run runs once right after it: %v", r.at)
	}
}

func TestThrottleCloseStopsTimer(t *testing.T) {
	r := newRuns(100 * time.Millisecond)
	r.t.mark()
	r.t.mark()
	r.t.close()
	r.clock.advance(time.Second)
	r.t.mark()
	if len(r.at) != 1 || r.clock.armed() != 0 {
		t.Fatalf("closed throttle ran %v, %d timers armed", r.at, r.clock.armed())
	}
}
