package acquisition

import (
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// pipeWorker is a test-only framed client (4-byte LE length + JSON) for the real-IED
// evidence test; the product uses stream C's internal/native client through StartWorker.
type pipeWorker struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	writeMu sync.Mutex
	mu      sync.Mutex
	pending map[string]chan *pyjson.Object
	next    atomic.Int64
	closing atomic.Bool
	done    chan struct{}
}

type timeoutError struct{ op string }

func (e timeoutError) Error() string { return "Native " + e.op + " request timed out" }
func (e timeoutError) Timeout() bool { return true }

func startPipe(executable string, onEvent func(*pyjson.Object)) (Worker, error) {
	cmd := exec.Command(executable)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	w := &pipeWorker{cmd: cmd, stdin: stdin, pending: map[string]chan *pyjson.Object{}, done: make(chan struct{})}
	go w.read(stdout, onEvent)
	return w, nil
}

func (w *pipeWorker) read(stdout io.Reader, onEvent func(*pyjson.Object)) {
	defer close(w.done)
	failure := "Worker stopped"
	for {
		var size [4]byte
		if _, err := io.ReadFull(stdout, size[:]); err != nil {
			failure = err.Error()
			break
		}
		payload := make([]byte, binary.LittleEndian.Uint32(size[:]))
		if _, err := io.ReadFull(stdout, payload); err != nil {
			failure = err.Error()
			break
		}
		decoded, err := pyjson.Decode(payload)
		message := pyv.Obj(decoded)
		if err != nil || message == nil {
			failure = "bad frame"
			break
		}
		if pyv.Has(message, "event") {
			onEvent(message)
			continue
		}
		w.mu.Lock()
		target := w.pending[pyv.GetStr(message, "id")]
		w.mu.Unlock()
		if target != nil {
			select {
			case target <- message:
			default:
			}
		}
	}
	w.mu.Lock()
	for _, target := range w.pending {
		select {
		case target <- pyv.O("ok", false, "error", failure):
		default:
		}
	}
	w.mu.Unlock()
	if !w.closing.Load() {
		onEvent(pyv.O("event", "worker_failed", "error", failure))
	}
}

func (w *pipeWorker) Request(op string, timeout time.Duration, args *pyjson.Object) (any, error) {
	id := fmt.Sprintf("%032x", w.next.Add(1))
	reply := make(chan *pyjson.Object, 1)
	w.mu.Lock()
	w.pending[id] = reply
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		delete(w.pending, id)
		w.mu.Unlock()
	}()
	frame := pyv.Merge(pyv.O("version", 1, "id", id, "op", op), args)
	payload, err := pyjson.Compact(frame)
	if err != nil {
		return nil, err
	}
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], uint32(len(payload)))
	w.writeMu.Lock()
	_, err = w.stdin.Write(append(size[:], payload...))
	w.writeMu.Unlock()
	if err != nil {
		return nil, apperr.RuntimeError(err.Error())
	}
	select {
	case message := <-reply:
		if !pyv.Truthy(pyv.Get(message, "ok")) {
			return nil, apperr.RuntimeError(pyv.PyStr(pyv.GetOr(message, "error", "Native request failed")))
		}
		return pyv.Get(message, "result"), nil
	case <-time.After(timeout):
		return nil, timeoutError{op: op}
	}
}

func (w *pipeWorker) Close() {
	w.closing.Store(true)
	_, _ = w.Request("stop", 3*time.Second, pyjson.NewObject())
	select {
	case <-w.done:
	case <-time.After(3 * time.Second):
		_ = w.cmd.Process.Kill()
	}
	_ = w.cmd.Wait()
}
