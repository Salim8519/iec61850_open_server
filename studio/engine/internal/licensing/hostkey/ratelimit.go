package hostkey

import (
	"sync"
	"time"
)

// Limiter is the rate limit of the proof service: at most PerHost requests from one address and
// Total requests overall within Window. A VM asks once at start and rarely again, so the limits
// are generous for people and tight for a program that hammers the port.
type Limiter struct {
	PerHost int
	Total   int
	Window  time.Duration
	Now     func() time.Time // nil means time.Now

	mu    sync.Mutex
	hosts map[string][]time.Time
	all   []time.Time
}

// DefaultLimiter allows 20 requests a minute from one address and 60 a minute overall.
func DefaultLimiter() *Limiter { return &Limiter{PerHost: 20, Total: 60, Window: time.Minute} }

func (l *Limiter) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

// keep drops the times that left the window.
func keep(times []time.Time, from time.Time) []time.Time {
	kept := times[:0]
	for _, at := range times {
		if at.After(from) {
			kept = append(kept, at)
		}
	}
	return kept
}

// Allow records a request from host and says whether it may be answered.
func (l *Limiter) Allow(host string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	from := now.Add(-l.Window)
	if l.hosts == nil {
		l.hosts = map[string][]time.Time{}
	}
	l.all = keep(l.all, from)
	mine := keep(l.hosts[host], from)
	if len(mine) >= l.PerHost || len(l.all) >= l.Total {
		l.hosts[host] = mine
		return false
	}
	l.hosts[host] = append(mine, now)
	l.all = append(l.all, now)
	if len(l.hosts) > 1024 { // a scan from many addresses must not grow the map without bound
		for name, times := range l.hosts {
			if len(keep(times, from)) == 0 {
				delete(l.hosts, name)
			}
		}
	}
	return true
}
