package worker

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Process is one running framed worker. Request returns the reply's "result"; native
// refusals are RuntimeErrors (503), a missed deadline a *Timeout.
type Process interface {
	Request(op string, timeout time.Duration, fields *pyjson.Object) (any, error)
	Alive() bool
	Close()
}

// Spawner starts a worker; onEvent receives every unsolicited frame in order on the reader.
type Spawner func(executable string, onEvent func(*pyjson.Object)) (Process, error)

// DefaultTimeout is WorkerProcess.request's default (10 s).
const DefaultTimeout = 10 * time.Second

type process struct {
	command  *exec.Cmd
	stdin    *os.File
	onEvent  func(*pyjson.Object)
	mu       sync.Mutex // writes and the pending table, like Python's RLock
	pending  map[string]chan *pyjson.Object
	closing  bool
	exited   chan struct{}
	readDone chan struct{}
	errDone  chan struct{}
}

// Spawn starts executable with no arguments, inherited environment and working folder,
// no console window, and its three pipes (WorkerProcess.__init__).
func Spawn(executable string, onEvent func(*pyjson.Object)) (Process, error) {
	if onEvent == nil {
		onEvent = func(*pyjson.Object) {}
	}
	childIn, parentIn, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	parentOut, childOut, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	parentErr, childErr, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	command := exec.Command(executable)
	command.Stdin, command.Stdout, command.Stderr = childIn, childOut, childErr
	hideWindow(command)
	startErr := command.Start()
	childIn.Close()
	childOut.Close()
	childErr.Close()
	if startErr != nil {
		parentIn.Close()
		parentOut.Close()
		parentErr.Close()
		return nil, startErr
	}
	p := &process{command: command, stdin: parentIn, onEvent: onEvent, pending: map[string]chan *pyjson.Object{},
		exited: make(chan struct{}), readDone: make(chan struct{}), errDone: make(chan struct{})}
	go func() { command.Wait(); close(p.exited) }()
	go p.read(parentOut)
	go p.diagnostics(parentErr)
	return p, nil
}

func (p *process) read(stdout *os.File) {
	defer close(p.readDone)
	defer stdout.Close()
	reader := bufio.NewReaderSize(stdout, 64*1024)
	var failure error
	for {
		message, err := readFrame(reader)
		if err != nil {
			failure = err
			break
		}
		if _, isEvent := message.Get("event"); isEvent {
			p.deliver(message)
			continue
		}
		id, _ := message.Get("id")
		key, _ := id.(string)
		p.mu.Lock()
		target := p.pending[key]
		p.mu.Unlock()
		if target != nil {
			offer(target, message)
		}
	}
	text := failure.Error()
	p.mu.Lock()
	for _, target := range p.pending {
		offer(target, pyjson.NewObject().Set("ok", false).Set("error", text))
	}
	closing := p.closing
	p.mu.Unlock()
	if !closing {
		p.deliver(pyjson.NewObject().Set("event", "worker_failed").Set("error", text))
	}
}

// offer hands a reply to a waiting request without ever blocking the reader.
func offer(target chan *pyjson.Object, message *pyjson.Object) {
	select {
	case target <- message:
	default:
	}
}

func (p *process) deliver(event *pyjson.Object) {
	defer func() { recover() }() // "Native event consumer failed" never stops the reader
	p.onEvent(event)
}

func (p *process) diagnostics(stderr *os.File) {
	defer close(p.errDone)
	defer stderr.Close()
	lines := bufio.NewReader(stderr)
	for {
		line, err := lines.ReadString('\n')
		if line != "" {
			text := strings.ToValidUTF8(line, "\xef\xbf\xbd")
			p.deliver(pyjson.NewObject().Set("event", "native_diagnostic").Set("message", pytext.Strip(text)))
		}
		if err != nil {
			return
		}
	}
}

// Alive is Popen.poll() is None.
func (p *process) Alive() bool {
	select {
	case <-p.exited:
		return false
	default:
		return true
	}
}

// Request sends {"version":1,"id","op",...fields} and waits for the reply.
func (p *process) Request(op string, timeout time.Duration, fields *pyjson.Object) (any, error) {
	message := pyjson.NewObject().Set("version", int64(1)).Set("id", newID()).Set("op", op)
	if fields != nil {
		for _, key := range fields.Keys() {
			value, _ := fields.Get(key)
			message.Set(key, value)
		}
	}
	id, _ := message.Get("id")
	response := make(chan *pyjson.Object, 1)
	p.mu.Lock()
	if !p.Alive() {
		p.mu.Unlock()
		return nil, apperr.RuntimeError("Native worker exited: " + strconv.Itoa(p.command.ProcessState.ExitCode()))
	}
	frame, err := encodeFrame(message)
	if err == nil {
		p.pending[id.(string)] = response
		_, err = p.stdin.Write(frame)
		if err != nil {
			delete(p.pending, id.(string))
		}
	}
	p.mu.Unlock()
	if err != nil {
		return nil, err
	}
	defer func() {
		p.mu.Lock()
		delete(p.pending, id.(string))
		p.mu.Unlock()
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case reply := <-response:
		if ok, _ := reply.Get("ok"); !pytext.Truthy(ok) {
			text := "Native request failed"
			if value, found := reply.Get("error"); found {
				text = pytext.Str(value)
			}
			return nil, apperr.RuntimeError(text)
		}
		result, _ := reply.Get("result")
		return result, nil
	case <-timer.C:
		return nil, &Timeout{Op: op}
	}
}

// Close stops the worker politely (stop, 3 s), then terminates it (3 s), then kills it.
func (p *process) Close() {
	p.mu.Lock()
	p.closing = true
	p.mu.Unlock()
	if p.Alive() {
		p.Request("stop", 3*time.Second, nil)
		if !waitFor(p.exited, 3*time.Second) {
			p.command.Process.Kill()
			if !waitFor(p.exited, 3*time.Second) {
				p.command.Process.Kill()
				<-p.exited
			}
		}
	}
	waitFor(p.readDone, time.Second)
	waitFor(p.errDone, time.Second)
	p.stdin.Close()
}

func waitFor(done chan struct{}, limit time.Duration) bool {
	select {
	case <-done:
		return true
	case <-time.After(limit):
		return false
	}
}

func newID() string {
	var raw [16]byte
	rand.Read(raw[:])
	raw[6] = raw[6]&0x0f | 0x40 // uuid4().hex shape
	raw[8] = raw[8]&0x3f | 0x80
	return hex.EncodeToString(raw[:])
}

// Timeout is Python's TimeoutError("Native <op> request timed out"): 503 like a
// RuntimeError, but callers that only catch RuntimeError let it pass (errors.As).
type Timeout struct{ Op string }

func (e *Timeout) Error() string { return "Native " + e.Op + " request timed out" }

// Unwrap lets the HTTP layer map it like a RuntimeError (503 with the same text).
func (e *Timeout) Unwrap() error { return apperr.RuntimeError(e.Error()) }
