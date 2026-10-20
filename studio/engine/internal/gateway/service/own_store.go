package service

import (
	"strings"
	"sync"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// ownStore holds the values the running gateway serves for own points (lane G, Go-only):
// set at Start from the saved list, changed by the page or by a command, always good quality.
type ownStore struct {
	mu      sync.Mutex
	values  map[string]ownValue
	persist sync.Mutex // one read-modify-write of the saved point list at a time
}

type ownValue struct {
	value  any
	timeMS int64 // 0 until the value changes: studio_gateway then stamps the send time
}

// reset serves the own monitoring points of a starting gateway with their saved values.
func (s *ownStore) reset(points []*pyjson.Object) {
	values := map[string]ownValue{}
	for _, point := range points {
		if gateway.IsOwn(point) && !strings.HasPrefix(text(point, "kind"), "command_") {
			values[text(point, "id")] = ownValue{value: get(point, "value")}
		}
	}
	s.mu.Lock()
	s.values = values
	s.mu.Unlock()
}

// set changes the value of a served point; false when the running gateway does not serve it.
func (s *ownStore) set(id string, value any, timeMS int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, served := s.values[id]; !served {
		return false
	}
	s.values[id] = ownValue{value: value, timeMS: timeMS}
	return true
}

// sample is what SCADA receives for an own point: its value with good quality.
func (s *ownStore) sample(id string) Sample {
	s.mu.Lock()
	current, served := s.values[id]
	s.mu.Unlock()
	return ownSample(id, current.value, current.timeMS, served)
}

// current is the served value when the gateway runs, else the saved one.
func (s *ownStore) current(id string, saved any) Sample {
	s.mu.Lock()
	current, served := s.values[id]
	s.mu.Unlock()
	if !served {
		return ownSample(id, saved, 0, true)
	}
	return ownSample(id, current.value, current.timeMS, true)
}

func ownSample(id string, value any, timeMS int64, ok bool) Sample {
	if value == nil {
		value = int64(0)
	}
	return Sample{ID: id, Value: value, Quality: int64(0), TimeMS: timeMS, SourceOK: ok}
}

// markOwn re-sends the own points whose value changed (at most every 100 ms, as IED values).
func (s *syncer) markOwn(ids []string) {
	s.mu.Lock()
	marked := false
	for _, id := range ids {
		if index, found := s.ownAt[id]; found {
			s.dirty[index], marked = true, true
		}
	}
	s.mu.Unlock()
	if marked {
		s.signal()
	}
}
