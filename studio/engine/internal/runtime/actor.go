package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"

	"studio-engine/internal/apperr"
	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// running is one IED in the run. A simulated IED has an actor goroutine that owns its worker
// pipe: every sample write, pulse and guarded update runs there one after another (Python's
// per-IED sample lock), so writes of different IEDs never wait for each other.
type running struct {
	model  *iedModel
	table  *valueTable
	ied    *pyjson.Object // the run's copy of the project IED
	worker *native.Ied    // nil for a real IED
	host   string
	port   int64

	bands atomic.Bool // the IED's "Use SCD deadbands" switch (deadbands.go, ADR-0118)

	jobs     chan func()
	mu       sync.Mutex
	closed   bool
	inflight sync.WaitGroup
}

func newRunning(model *iedModel, table *valueTable, ied *pyjson.Object) *running {
	item := &running{model: model, table: table, ied: ied}
	item.bands.Store(pyval.Get(ied, "scd_deadbands") == true)
	return item
}

func (r *running) simulated() bool { return r.model.role == "simulated" }

func (r *running) startActor() {
	r.jobs = make(chan func(), 16)
	go func() {
		for job := range r.jobs {
			job()
		}
	}()
}

var errNotRunning = apperr.ValueError("IED is not running")

// do runs fn on the IED's actor and waits for it; a stopped actor answers "IED is not running".
func (r *running) do(fn func() error) (err error) {
	r.mu.Lock()
	if r.closed || r.jobs == nil {
		r.mu.Unlock()
		return errNotRunning
	}
	r.inflight.Add(1)
	r.mu.Unlock()
	defer r.inflight.Done()
	done := make(chan struct{})
	r.jobs <- func() {
		defer close(done)
		defer func() {
			if recovered := recover(); recovered != nil {
				err = panicError(recovered)
			}
		}()
		err = fn()
	}
	<-done
	return err
}

// stopActor refuses new work, waits for queued work and ends the goroutine.
func (r *running) stopActor() {
	r.mu.Lock()
	if r.closed || r.jobs == nil {
		r.closed = true
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.mu.Unlock()
	r.inflight.Wait()
	close(r.jobs)
}

// panicError turns a recovered panic into Python's generic 500 (an error that is not apperr).
func panicError(recovered any) error {
	if err, ok := recovered.(error); ok {
		return fmt.Errorf("internal error: %w", err)
	}
	return fmt.Errorf("internal error: %v", recovered)
}

// errorText is str(error) for the event messages Python built from exceptions.
func errorText(err error) string {
	if err == nil {
		return "None"
	}
	return err.Error()
}
