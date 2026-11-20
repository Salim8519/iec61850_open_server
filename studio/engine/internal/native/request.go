package native

import (
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Request sends {"version":1,"id","op",...fields} and returns the reply's result (nil for a
// reply without one), with WorkerProcess.request's errors: a dead worker and an ok:false
// reply are RuntimeErrors (HTTP 503), a timeout is a *TimeoutError (also 503; Timeout()
// reports true), a frame that cannot be written is a ValueError (400). The argument order is
// the one acquisition.Worker and tools/worker.Process declare.
func (w *Worker) Request(op string, timeout time.Duration, fields *pyjson.Object) (any, error) {
	return w.call(op, fields, timeout)
}

// call is Request with the typed clients' argument order.
func (w *Worker) call(op string, fields *pyjson.Object, timeout time.Duration) (any, error) {
	id := newRequestID()
	message := pyjson.NewObject().Set("version", int64(1)).Set("id", id).Set("op", op)
	if fields != nil {
		for _, key := range fields.Keys() {
			value, _ := fields.Get(key)
			message.Set(key, value)
		}
	}
	response := make(chan *pyjson.Object, 1)
	w.mu.Lock()
	if w.exited {
		code := w.exitCode
		w.mu.Unlock()
		return nil, exitedError(code)
	}
	w.pending[id] = response
	w.mu.Unlock()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	frame, err := EncodeFrame(message)
	if err == nil {
		err = w.write(op, frame, timer.C)
	}
	if err != nil {
		w.mu.Lock()
		delete(w.pending, id)
		w.mu.Unlock()
		if frameError, ok := err.(*FrameError); ok {
			return nil, apperr.ValueError(frameError.Message)
		}
		if IsTimeout(err) {
			return nil, err
		}
		return nil, apperr.RuntimeError(err.Error())
	}
	defer func() {
		w.mu.Lock()
		delete(w.pending, id)
		w.mu.Unlock()
	}()
	select {
	case reply := <-response:
		return replyResult(reply)
	case <-timer.C:
		return nil, newTimeout(op)
	}
}

// write sends one frame within the request's deadline; the pipe has its own lock so the
// dispatcher never waits behind a write. A worker that does not take the frame in time is
// killed: it may be half written, and a late command must never run.
func (w *Worker) write(op string, frame []byte, deadline <-chan time.Time) error {
	written := make(chan error, 1)
	go func() {
		w.writeMu.Lock()
		_, err := w.stdin.Write(frame)
		w.writeMu.Unlock()
		written <- err
	}()
	select {
	case err := <-written:
		return err
	case <-deadline:
		_ = w.command.Process.Kill()
		return newTimeout(op)
	}
}

func replyResult(reply *pyjson.Object) (any, error) {
	if ok, _ := reply.Get("ok"); !truthy(ok) {
		message, present := reply.Get("error")
		if !present {
			return nil, apperr.RuntimeError("Native request failed")
		}
		return nil, apperr.RuntimeError(pyStr(message))
	}
	result, _ := reply.Get("result")
	return result, nil
}

// Close stops the worker as WorkerProcess.close does: a "stop" request (3 s), a 3 s wait,
// then termination, and waits for the reader to drain. It never reports worker_failed.
func (w *Worker) Close() {
	w.mu.Lock()
	w.closing = true
	exited := w.exited
	w.mu.Unlock()
	if !exited {
		_, _ = w.call("stop", nil, 3*time.Second)
		select {
		case <-w.done:
		case <-time.After(3 * time.Second):
			_ = w.command.Process.Kill()
			select {
			case <-w.done:
			case <-time.After(3 * time.Second):
				_ = w.command.Process.Kill()
				<-w.done
			}
		}
	}
	_ = w.stdin.Close()
	select {
	case <-w.drained:
	case <-time.After(time.Second):
	}
}

// Kill ends the process at once, without the "stop" request, and waits for its pipes like
// Close. For a worker busy with a request it cannot interrupt (a connect toward a device that
// does not answer), where "stop" would wait behind that request (change 0223).
func (w *Worker) Kill() {
	w.mu.Lock()
	w.closing = true
	w.mu.Unlock()
	_ = w.command.Process.Kill()
	select {
	case <-w.done:
	case <-time.After(3 * time.Second):
	}
	_ = w.stdin.Close()
	select {
	case <-w.drained:
	case <-time.After(time.Second):
	}
}

// Done is closed when the process has exited and its pipes are drained.
func (w *Worker) Done() <-chan struct{} { return w.done }
