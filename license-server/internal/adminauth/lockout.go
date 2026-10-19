package adminauth

import (
	"sync"
	"time"
)

// Lockout stops sign-in for everyone after too many wrong tries in a window, so an
// attacker spreading guesses over many addresses is still stopped. The per-address
// limit is separate (the api package's rate limiter).
type Lockout struct {
	mu          sync.Mutex
	limit       int
	window      time.Duration
	duration    time.Duration
	failures    []time.Time
	lockedUntil time.Time
}

// NewLockout locks for duration after limit failures within window.
func NewLockout(limit int, window, duration time.Duration) *Lockout {
	return &Lockout{limit: limit, window: window, duration: duration}
}

// Locked reports whether sign-in is closed and until when.
func (l *Lockout) Locked(now time.Time) (bool, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return now.Before(l.lockedUntil), l.lockedUntil
}

// Fail records a wrong try and returns true when this try started a lock.
func (l *Lockout) Fail(now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	recent := l.failures[:0]
	for _, at := range l.failures {
		if now.Sub(at) < l.window {
			recent = append(recent, at)
		}
	}
	l.failures = append(recent, now)
	if len(l.failures) >= l.limit {
		l.failures = nil
		l.lockedUntil = now.Add(l.duration)
		return true
	}
	return false
}

// Succeed clears the failure count after a correct sign-in.
func (l *Lockout) Succeed() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures = nil
}
