package service

import (
	"time"

	"studio-engine/internal/tools/worker"
)

// Connect no longer waits for the interrogation it starts (ADR-0142): the GI confirmation, its
// end and the counter interrogation could keep the Connect button busy for up to a minute while
// the values were already arriving as events. What needed the finished interrogation waits for
// it here instead: the protocol converter maps the received points only after it (as when it
// started after a waiting Connect), and Disconnect does not queue behind it.

// askLimit is the longest the converter waits: the interrogate request's own deadline.
const askLimit = 60 * time.Second

func (m *Master) beginAsk() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.asking == 0 {
		m.askIdle = make(chan struct{})
	}
	m.asking++
}

func (m *Master) endAsk() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.asking--
	if m.asking == 0 {
		close(m.askIdle)
	}
}

// waitAsked returns once no interrogation is in flight, or after limit.
func (m *Master) waitAsked(limit time.Duration) {
	m.mu.Lock()
	idle, busy := m.askIdle, m.asking > 0
	m.mu.Unlock()
	if !busy {
		return
	}
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case <-idle:
	case <-timer.C:
	}
}

// end stops a worker that Disconnect took over. studio_master answers one request at a time,
// so "stop" would wait behind an interrogation still in flight (up to 60 s); the worker is
// ended at once instead, which closes the link as "stop" does.
func (m *Master) end(process worker.Process) {
	m.mu.Lock()
	busy := m.asking > 0
	m.mu.Unlock()
	if killer, ok := process.(interface{ Kill() }); ok && busy {
		killer.Kill()
	} else {
		process.Request("stop", 10*time.Second, nil)
	}
	process.Close()
}
