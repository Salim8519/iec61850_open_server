package runtime

import (
	"sync"
	"time"
)

// typedSelects remembers which typed controls ScadaBench selected and until when, so an
// Operate after an expired Select says so instead of selecting again behind the user's back.
// An Operate without any Select keeps working in one step (select + operate), as today.
type typedSelects struct {
	mu    sync.Mutex
	until map[string]time.Time
	span  map[string]int64
}

func selectKey(iedID, ref string) string { return iedID + "|" + ref }

func (s *typedSelects) remember(iedID, ref string, timeoutMs int64, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.until == nil {
		s.until, s.span = map[string]time.Time{}, map[string]int64{}
	}
	key := selectKey(iedID, ref)
	s.until[key] = now.Add(time.Duration(timeoutMs) * time.Millisecond)
	s.span[key] = timeoutMs
}

// take removes the record: selected says a Select was made; expired that it ran out.
func (s *typedSelects) take(iedID, ref string, now time.Time) (selected, expired bool, timeoutMs int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := selectKey(iedID, ref)
	until, found := s.until[key]
	if !found {
		return false, false, 0
	}
	timeoutMs = s.span[key]
	delete(s.until, key)
	delete(s.span, key)
	return true, !now.Before(until), timeoutMs
}

func (s *typedSelects) forget(iedID, ref string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.until, selectKey(iedID, ref))
	delete(s.span, selectKey(iedID, ref))
}
