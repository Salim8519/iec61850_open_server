package acquisition

import (
	"errors"
	"sync"
	"time"

	"studio-engine/internal/pyjson"
)

// A live client answers one request at a time. A "disconnect" sent while it is still
// connecting to (15 s) or reading the data model of (120 s) a device that does not answer
// waited behind that request, so Stop and Disconnect took that long (change 0223). Until the
// session reserves its first report block there is nothing to give back (ADR-0067), so a
// busy client is killed instead; once a block may be reserved, "disconnect" is sent as before.

var errStopping = errors.New("stopping")

// busyWorker counts the requests in flight on a session's live client.
type busyWorker struct {
	Worker
	mu   sync.Mutex
	busy int
}

func (w *busyWorker) Request(op string, timeout time.Duration, args *pyjson.Object) (any, error) {
	w.mu.Lock()
	w.busy++
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		w.busy--
		w.mu.Unlock()
	}()
	return w.Worker.Request(op, timeout, args)
}

func (w *busyWorker) idle() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.busy == 0
}

type killer interface{ Kill() }

// willReserve notes that the session is about to reserve a report block; false when Stop has
// begun (the session gives up instead, so a killed client never leaves a block reserved).
func (s *RealSession) willReserve() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.isStopping() {
		return false
	}
	s.reserved = true
	return true
}

// closeProcess ends the session's live client: killed when it is busy and holds no report
// block, otherwise "disconnect" (every block given back) and close.
func (s *RealSession) closeProcess() {
	s.mu.Lock()
	process, reserved := s.process, s.reserved
	s.process = nil
	s.mu.Unlock()
	if process == nil {
		return
	}
	if tracked, ok := process.(*busyWorker); ok && !reserved && !tracked.idle() {
		if kill, ok := tracked.Worker.(killer); ok {
			kill.Kill()
			return
		}
	}
	_, _ = process.Request("disconnect", 15*time.Second, pyjson.NewObject())
	process.Close()
}
