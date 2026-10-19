package native

import (
	"errors"
	"fmt"

	"studio-engine/internal/apperr"
)

// TimeoutError is Python's TimeoutError from WorkerProcess.request: "Native <op> request
// timed out". It unwraps to an apperr Runtime error, so the HTTP layer answers 503 with the
// same body, and implements Timeout() bool like net.Error, so acquisition.IsTimeout and
// similar checks recognise it without importing this package.
type TimeoutError struct {
	Op    string
	cause *apperr.Error
}

func newTimeout(op string) *TimeoutError {
	return &TimeoutError{Op: op, cause: apperr.RuntimeError(fmt.Sprintf("Native %s request timed out", op))}
}

func (e *TimeoutError) Error() string { return e.cause.Message }

// Timeout always reports true.
func (e *TimeoutError) Timeout() bool { return true }

// Unwrap gives the apperr Runtime error (errors.As(err, &*apperr.Error) finds it).
func (e *TimeoutError) Unwrap() error { return e.cause }

// IsTimeout reports whether err is (or wraps) a request timeout.
func IsTimeout(err error) bool {
	var timeout interface{ Timeout() bool }
	return errors.As(err, &timeout) && timeout.Timeout()
}

// Alive reports whether the process is still running (tools/worker.Process declares it).
func (w *Worker) Alive() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return !w.exited
}
