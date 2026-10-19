//go:build !windows

package uplink

import "sync"

// signals outside Windows: Npcap is never available there, so only stop and send
// wake the thread (the receive event is ignored).
type signals struct {
	once sync.Once
	stop chan struct{}
	send chan struct{}
}

func newSignals() (*signals, error) {
	return &signals{stop: make(chan struct{}), send: make(chan struct{}, 1)}, nil
}

func (s *signals) kick() {
	select {
	case s.send <- struct{}{}:
	default:
	}
}

func (s *signals) halt() { s.once.Do(func() { close(s.stop) }) }

func (s *signals) wait(uintptr) error {
	select {
	case <-s.stop:
	case <-s.send:
	}
	return nil
}

func (s *signals) release() {}
