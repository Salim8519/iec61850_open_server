package native

import (
	"fmt"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Request sends {"version":1,"id","op",...fields} and returns the reply's result (nil for a
// reply without one), with WorkerProcess.request's errors: a dead worker, an ok:false reply
// and a timeout are RuntimeError/TimeoutError (HTTP 503); a frame that cannot be written is
// a ValueError (400).
func (w *Worker) Request(op string, fields *pyjson.Object, timeout time.Duration) (any, error) {
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
	frame, err := EncodeFrame(message)
	if err == nil {
		// The pipe has its own lock so the dispatcher never waits behind a write.
		w.writeMu.Lock()
		_, err = w.stdin.Write(frame)
		w.writeMu.Unlock()
	}
	if err != nil {
		w.mu.Lock()
		delete(w.pending, id)
		w.mu.Unlock()
		if frameError, ok := err.(*FrameError); ok {
			return nil, apperr.ValueError(frameError.Message)
		}
		return nil, apperr.RuntimeError(err.Error())
	}
	defer func() {
		w.mu.Lock()
		delete(w.pending, id)
		w.mu.Unlock()
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case reply := <-response:
		return replyResult(reply)
	case <-timer.C:
		return nil, apperr.RuntimeError(fmt.Sprintf("Native %s request timed out", op))
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
		_, _ = w.Request("stop", nil, 3*time.Second)
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

// Done is closed when the process has exited and its pipes are drained.
func (w *Worker) Done() <-chan struct{} { return w.done }
