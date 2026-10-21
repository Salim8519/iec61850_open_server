// Package push is the Go-only push stream (speed plan phase 2, lanes G1-G3): one SSE stream
// per browser tab carrying the unchanged bus events of /api/v1/events plus topic frames that
// replace the frontend's polling. A topic frame is the exact body of the polled GET route
// ("full"), a keyed delta or an append, with a per-topic revision. Topics are computed only
// while a mounted, visible view subscribes; an idle server runs no timers.
package push

import "time"

// Clock is the time source of the coalescing timers (a fake one in tests).
type Clock interface {
	Now() time.Time
	// AfterFunc runs f on its own goroutine after d, unless the timer is stopped first.
	AfterFunc(d time.Duration, f func()) Timer
}

// Timer is a pending AfterFunc call.
type Timer interface {
	// Stop prevents the call; false when it already ran or was stopped.
	Stop() bool
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now() }

func (wallClock) AfterFunc(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) }

// WallClock is the real clock.
var WallClock Clock = wallClock{}
