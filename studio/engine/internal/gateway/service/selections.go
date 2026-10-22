package service

import (
	"sync"
	"time"
)

// selections remember which points the IED selected for a SCADA select (ADR-0107), so the
// execute that follows completes that selection and a cancel releases it. A selection ends
// with its execute or cancel, or when the IED's select time runs out.
type selections struct {
	mu    sync.Mutex
	until map[string]time.Time
}

// defaultSelect is how long a selection lasts when the IED does not say (sboTimeout 30 s).
const defaultSelect = 30 * time.Second

func (s *selections) put(pointID string, expiresMs int) {
	lasts := time.Duration(expiresMs) * time.Millisecond
	if lasts <= 0 {
		lasts = defaultSelect
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.until == nil {
		s.until = map[string]time.Time{}
	}
	s.until[pointID] = time.Now().Add(lasts)
}

// has reports a selection that has not run out.
func (s *selections) has(pointID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, found := s.until[pointID]
	return found && time.Now().Before(until)
}

// take ends a selection and reports whether one was still running.
func (s *selections) take(pointID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	until, found := s.until[pointID]
	delete(s.until, pointID)
	return found && time.Now().Before(until)
}
