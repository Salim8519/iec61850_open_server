package recovery

import (
	"time"

	"studio-engine/internal/pyjson"
)

// Background is BackgroundRecovery: startup recovery runs on its own goroutine so the app
// window can appear at once; engineering requests wait for it (api/startup_gate.py). A
// recovery that stopped to protect the autosave keeps its error for every waiting request.
type Background struct {
	done   chan struct{}
	result *pyjson.Object
	err    error
}

// StartBackground runs recover (normally a closure over Fold + RecoverStartup) once.
// logError (may be nil) receives the error, like logging.exception("Startup recovery stopped").
func StartBackground(recover func() (*pyjson.Object, error), logError func(error)) *Background {
	b := &Background{done: make(chan struct{})}
	go func() {
		defer close(b.done)
		b.result, b.err = recover()
		if b.err != nil && logError != nil {
			logError(b.err)
		}
	}()
	return b
}

// Wait reports whether recovery finished (successfully or not) within timeout; a negative
// timeout waits forever (Python's wait(None)).
func (b *Background) Wait(timeout time.Duration) bool {
	if timeout < 0 {
		<-b.done
		return true
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-b.done:
		return true
	case <-timer.C:
		return false
	}
}

// Done is closed when recovery finished.
func (b *Background) Done() <-chan struct{} { return b.done }

// Result returns recovery.result and recovery.error; both are nil until Done is closed.
func (b *Background) Result() (*pyjson.Object, error) {
	select {
	case <-b.done:
		return b.result, b.err
	default:
		return nil, nil
	}
}
