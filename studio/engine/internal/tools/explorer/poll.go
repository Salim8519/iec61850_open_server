package explorer

import (
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// SetWatch replaces the watch list (deduplicated by ref and fc) and the poll interval.
func (s *Service) SetWatch(items []*pyjson.Object, intervalMS any) (*pyjson.Object, error) {
	defer s.changed()
	var clean []*pyjson.Object
	seen := map[[2]string]bool{}
	for _, item := range items {
		key := [2]string{pytext.Str(get(item, "ref")), pytext.Str(get(item, "fc"))}
		if seen[key] {
			continue
		}
		seen[key] = true
		kind := ""
		if value, found := item.Get("type"); found {
			kind = pytext.Str(value)
		}
		clean = append(clean, pyjson.NewObject().Set("ref", key[0]).Set("fc", key[1]).Set("type", kind))
	}
	if len(clean) > MaxWatch {
		return nil, apperr.ValueError("The watch list holds at most 200 attributes")
	}
	if intervalMS != nil {
		number, ok := pytext.Int64(intervalMS)
		if !ok || !pytext.IsIntOrBool(intervalMS) || number < MinIntervalMS || number > MaxIntervalMS {
			return nil, apperr.ValueError("Choose a poll interval between 0.5 and 60 seconds")
		}
		s.mu.Lock()
		s.interval = number
		s.mu.Unlock()
	}
	s.mu.Lock()
	s.watch = clean
	s.mu.Unlock()
	if s.memory != nil {
		if err := s.remember(); err != nil {
			return nil, err
		}
	}
	return s.Status(), nil
}

// StartPolling reads the watch list every interval in the background.
func (s *Service) StartPolling() (*pyjson.Object, error) {
	defer s.changed()
	s.mu.Lock()
	empty := len(s.watch) == 0
	s.mu.Unlock()
	if empty {
		return nil, apperr.ValueError("Add attributes to the watch list first")
	}
	if _, err := s.request("hello", requestTimeout, nil); err != nil {
		return nil, err
	}
	s.StopPolling()
	stop, done := make(chan struct{}), make(chan struct{})
	s.mu.Lock()
	s.pollStop, s.pollDone, s.polling = stop, done, true
	s.mu.Unlock()
	go s.poll(stop, done)
	return s.Status(), nil
}

func (s *Service) poll(stop, done chan struct{}) {
	defer close(done)
	for !stopped(stop) {
		started := time.Now()
		s.mu.Lock()
		items := append([]*pyjson.Object(nil), s.watch...)
		interval := time.Duration(s.interval) * time.Millisecond
		s.mu.Unlock()
		if len(items) > 0 {
			if _, err := s.Read(items); err != nil {
				s.setMessage("Polling paused: " + err.Error())
				break
			}
		}
		wait := interval - time.Since(started)
		if wait < 50*time.Millisecond {
			wait = 50 * time.Millisecond
		}
		select {
		case <-stop:
		case <-time.After(wait):
		}
	}
	s.mu.Lock()
	s.polling = false
	s.mu.Unlock()
	s.changed()
}

func stopped(stop chan struct{}) bool {
	select {
	case <-stop:
		return true
	default:
		return false
	}
}

// StopPolling ends the poll loop and waits up to 10 s for it.
func (s *Service) StopPolling() *pyjson.Object {
	defer s.changed()
	s.mu.Lock()
	stop, done := s.pollStop, s.pollDone
	s.pollDone = nil
	s.mu.Unlock()
	closeOnce(stop)
	if done != nil {
		select {
		case <-done:
		case <-time.After(10 * time.Second):
		}
	}
	s.mu.Lock()
	s.polling = false
	s.mu.Unlock()
	return s.Status()
}
