package guard

import (
	"errors"
	"time"

	"studio-goose-engine/internal/frame"
)

// Limits caps how many frames per second one IED may send outside, so a
// runaway simulation can never flood a real network.
type Limits struct {
	perSecond int
	windows   map[string]*window
}

type window struct {
	start time.Time
	count int
}

// NewLimits returns a per-IED cap. Callers hold the guard lock.
func NewLimits(perSecond int) *Limits {
	return &Limits{perSecond: perSecond, windows: map[string]*window{}}
}

// Allow counts one outward frame for the IED in the current one-second window.
func (l *Limits) Allow(ied string, now time.Time) bool {
	w := l.windows[ied]
	if w == nil || now.Sub(w.start) >= time.Second {
		w = &window{start: now}
		l.windows[ied] = w
	}
	w.count++
	return w.count <= l.perSecond
}

// ValidOutbound refuses anything that is not a plain multicast GOOSE frame.
func ValidOutbound(info frame.Info) error {
	if info.Size == 0 || info.Size > frame.MaxFrame {
		return errors.New("frame size is outside Ethernet limits")
	}
	if !info.Dst.Multicast() {
		return errors.New("GOOSE destination is not multicast")
	}
	if info.GocbRef == "" {
		return errors.New("GOOSE frame has no gocbRef")
	}
	return nil
}
