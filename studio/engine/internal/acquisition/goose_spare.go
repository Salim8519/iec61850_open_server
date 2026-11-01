package acquisition

import (
	"os"
	"time"
)

// spareEngine is a fresh studio_goose process, started and greeted while no Run needs it, so
// the next Run does not wait for a process start in the middle of its own busy start
// (ADR-0133: about one second of a Run on a busy PC). It is a new process every time, exactly
// what that Run would have started; only the moment differs. A spare is never used when the
// program file changed after it started (a rebuilt or updated engine is started anew).
type spareEngine struct {
	process   Worker
	stamp     fileStamp
	preparing chan struct{} // open while a spare is being started; nil otherwise
	closed    bool          // the application is shutting down
}

type fileStamp struct {
	size     int64
	modified time.Time
}

func stampOf(path string) (fileStamp, bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return fileStamp{}, false
	}
	return fileStamp{info.Size(), info.ModTime()}, true
}

// Prepare starts a spare engine in the background unless one runs, waits or is starting.
// A refused or failed start leaves no spare: the next Run starts the engine as before.
func (e *GooseEngine) Prepare() {
	e.mu.Lock()
	if e.spare.closed || e.spare.process != nil || e.spare.preparing != nil || e.process != nil {
		e.mu.Unlock()
		return
	}
	stamp, found := stampOf(e.executable)
	if !found {
		e.mu.Unlock()
		return
	}
	done := make(chan struct{})
	e.spare.preparing = done
	e.mu.Unlock()
	go func() {
		defer close(done)
		process, err := e.greet()
		e.mu.Lock()
		e.spare.preparing = nil
		keep := err == nil && !e.spare.closed && e.spare.process == nil
		if keep {
			e.spare.process, e.spare.stamp = process, stamp
		}
		e.mu.Unlock()
		if err == nil && !keep {
			process.Close()
		}
	}()
}

// takeSpare hands the spare to the Run (waiting for one that is starting), or nil when there
// is none or its program file changed since it started.
func (e *GooseEngine) takeSpare() Worker {
	e.mu.Lock()
	preparing := e.spare.preparing
	e.mu.Unlock()
	if preparing != nil {
		<-preparing
	}
	e.mu.Lock()
	process, stamp := e.spare.process, e.spare.stamp
	e.spare.process = nil
	e.mu.Unlock()
	if process == nil {
		return nil
	}
	if now, found := stampOf(e.executable); !found || now != stamp {
		process.Close()
		return nil
	}
	return process
}

// Discard closes the spare and starts no other one (application shutdown).
func (e *GooseEngine) Discard() {
	e.mu.Lock()
	e.spare.closed = true
	process, preparing := e.spare.process, e.spare.preparing
	e.spare.process = nil
	e.mu.Unlock()
	if preparing != nil {
		<-preparing // the starter sees closed and closes its own process
	}
	if process != nil {
		process.Close()
	}
}
