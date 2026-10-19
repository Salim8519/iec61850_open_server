package native

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// DefaultTimeout is WorkerProcess.request's default (10 s).
const DefaultTimeout = 10 * time.Second

// EventFunc receives every unsolicited frame (a frame with an "event" key), the stderr
// lines as native_diagnostic and the final worker_failed, one at a time and in pipe order.
type EventFunc func(event *pyjson.Object)

// Worker is one native process with Python's WorkerProcess semantics: requests carry
// {"version":1,"id","op",...}; replies are matched by id; events are handed to the consumer
// in order, before any reply that followed them on the pipe; a reader never blocks on the
// consumer.
type Worker struct {
	command *exec.Cmd
	stdin   io.WriteCloser
	onEvent EventFunc
	inbox   *fifo[inboxItem]

	mu       sync.Mutex
	writeMu  sync.Mutex
	pending  map[string]chan *pyjson.Object
	closing  bool
	exited   bool
	exitCode int
	done     chan struct{}
	drained  chan struct{}
}

type inboxItem struct {
	frame   *pyjson.Object
	event   bool
	end     bool
	failure any // nil or string: the reader's failure text
}

// Start launches executable with no arguments, inheriting the environment and working
// folder (processes-1.md §3), hidden and inside the server's kill-on-close job.
func Start(executable string, onEvent EventFunc) (*Worker, error) {
	if onEvent == nil {
		onEvent = func(*pyjson.Object) {}
	}
	command := exec.Command(executable)
	prepareCommand(command)
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		return nil, err
	}
	_ = adopt(command.Process.Pid)
	w := &Worker{command: command, stdin: stdin, onEvent: onEvent, inbox: newFifo[inboxItem](),
		pending: map[string]chan *pyjson.Object{}, done: make(chan struct{}), drained: make(chan struct{})}
	stdoutDone, stderrDone := make(chan struct{}), make(chan struct{})
	go w.wait(stdoutDone, stderrDone)
	go w.read(stdout, stdoutDone)
	go w.diagnostics(stderr, stderrDone)
	go w.dispatch()
	return w, nil
}

// wait reaps the process once both pipes are drained (exec.Cmd.Wait requires it).
func (w *Worker) wait(stdoutDone, stderrDone chan struct{}) {
	<-stdoutDone
	<-stderrDone
	err := w.command.Wait()
	w.mu.Lock()
	w.exited = true
	w.exitCode = w.command.ProcessState.ExitCode()
	w.mu.Unlock()
	_ = err
	close(w.done)
}

func (w *Worker) read(stdout io.Reader, done chan struct{}) {
	defer close(done)
	reader := bufio.NewReaderSize(stdout, 64*1024)
	var failure any
	for {
		message, err := ReadFrame(reader)
		if err != nil {
			failure = err.Error()
			break
		}
		frame, ok := message.(*pyjson.Object)
		if !ok {
			failure = "'" + typeName(message) + "' object has no attribute 'get'"
			break
		}
		_, isEvent := frame.Get("event")
		w.inbox.push(inboxItem{frame: frame, event: isEvent})
	}
	w.inbox.push(inboxItem{end: true, failure: failure})
}

// diagnostics turns each stderr line into a native_diagnostic event (worker_process.py:53-55).
func (w *Worker) diagnostics(stderr io.Reader, done chan struct{}) {
	defer close(done)
	reader := bufio.NewReader(stderr)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			event := pyjson.NewObject().Set("event", "native_diagnostic").
				Set("message", pyStrip(decodeReplace(line)))
			w.inbox.push(inboxItem{frame: event, event: true})
		}
		if err != nil {
			return
		}
	}
}

func (w *Worker) dispatch() {
	defer close(w.drained)
	for {
		item, ok := w.inbox.pop()
		if !ok {
			return
		}
		switch {
		case item.end:
			w.finish(item.failure)
			w.inbox.close()
		case item.event:
			w.deliver(item.frame)
		default:
			id, _ := item.frame.Get("id")
			key, _ := id.(string)
			w.mu.Lock()
			target := w.pending[key]
			w.mu.Unlock()
			if target != nil {
				target <- item.frame
			}
		}
	}
}

// finish fails every pending request, then reports worker_failed unless the worker is closing.
func (w *Worker) finish(failure any) {
	text := "Worker stopped"
	if failure != nil {
		text = failure.(string)
	}
	w.mu.Lock()
	for _, target := range w.pending {
		select {
		case target <- pyjson.NewObject().Set("ok", false).Set("error", text):
		default:
		}
	}
	closing := w.closing
	w.mu.Unlock()
	if !closing {
		w.deliver(pyjson.NewObject().Set("event", "worker_failed").Set("error", failure))
	}
}

func (w *Worker) deliver(event *pyjson.Object) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("Native event consumer failed: %v", recovered)
		}
	}()
	w.onEvent(event)
}

func newRequestID() string {
	var raw [16]byte
	_, _ = rand.Read(raw[:])
	raw[6] = raw[6]&0x0f | 0x40 // uuid4 hex, as uuid4().hex
	raw[8] = raw[8]&0x3f | 0x80
	return hex.EncodeToString(raw[:])
}

func typeName(value any) string {
	switch value.(type) {
	case []any:
		return "list"
	case string:
		return "str"
	case bool:
		return "bool"
	case float64:
		return "float"
	case nil:
		return "NoneType"
	}
	return "int"
}

// Exited reports whether the process ended, with its exit code.
func (w *Worker) Exited() (bool, int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.exited, w.exitCode
}

// Pid is the worker's process id (tests and diagnostics only).
func (w *Worker) Pid() int { return w.command.Process.Pid }

func exitedError(code int) error {
	return apperr.RuntimeError(fmt.Sprintf("Native worker exited: %d", code))
}
