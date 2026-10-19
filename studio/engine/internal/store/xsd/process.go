package xsd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// process is one running studio_xsd: framed requests with ids, one reader goroutine
// that routes replies to their callers, and a bounded tail of stderr for diagnostics.
type process struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	writeMu sync.Mutex
	nextID  atomic.Uint64
	mu      sync.Mutex
	pending map[string]chan response
	done    chan struct{} // closed once the helper's output ended and it was reaped
	failure string        // why it ended; read after done is closed
	stderr  *tail
}

var errTimedOut = errors.New("request timed out")

func startProcess(path string) (*process, error) {
	cmd := exec.Command(path)
	hideWindow(cmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	p := &process{cmd: cmd, stdin: stdin, pending: map[string]chan response{},
		done: make(chan struct{}), stderr: newTail(2048)}
	cmd.Stderr = p.stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go p.read(stdout)
	return p, nil
}

func (p *process) read(stdout io.Reader) {
	failure := "the helper closed its output"
	for {
		payload, err := readFrame(stdout)
		if err != nil {
			if err != io.EOF {
				failure = err.Error()
			}
			break
		}
		var message response
		if err := json.Unmarshal(payload, &message); err != nil {
			failure = "unreadable reply: " + err.Error()
			break
		}
		p.mu.Lock()
		target := p.pending[message.ID]
		delete(p.pending, message.ID)
		p.mu.Unlock()
		if target != nil {
			target <- message
		}
	}
	_ = p.cmd.Process.Kill() // a helper with a broken stream is never reused
	waitErr := p.cmd.Wait()
	if waitErr != nil {
		failure = fmt.Sprintf("%s; %v", failure, waitErr)
	}
	if last := p.stderr.lastLine(); last != "" {
		failure = fmt.Sprintf("%s; %s", failure, last)
	}
	p.mu.Lock()
	p.failure = failure
	p.pending = nil
	p.mu.Unlock()
	close(p.done)
}

func (p *process) alive() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

// request sends one operation and waits for its reply, the helper's end, or timeout.
func (p *process) request(message request, timeout time.Duration) (json.RawMessage, error) {
	message.Version = protocolVersion
	message.ID = strconv.FormatUint(p.nextID.Add(1), 10)
	reply := make(chan response, 1)
	p.mu.Lock()
	if p.pending == nil {
		p.mu.Unlock()
		return nil, p.ended()
	}
	p.pending[message.ID] = reply
	p.mu.Unlock()
	p.writeMu.Lock()
	err := writeFrame(p.stdin, message)
	p.writeMu.Unlock()
	if err != nil {
		p.forget(message.ID)
		p.kill()
		<-p.done
		return nil, p.ended()
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case answer := <-reply:
		if !answer.OK {
			return nil, errHelper(answer.Error)
		}
		return answer.Result, nil
	case <-p.done:
		return nil, p.ended()
	case <-timer.C:
		p.forget(message.ID)
		return nil, errTimedOut
	}
}

func (p *process) forget(id string) {
	p.mu.Lock()
	if p.pending != nil {
		delete(p.pending, id)
	}
	p.mu.Unlock()
}

// ended is the crash error after done is closed.
func (p *process) ended() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return errCrashed(p.failure)
}

func (p *process) kill() {
	_ = p.cmd.Process.Kill()
}

// stop asks the helper to exit, then kills it if it does not within the grace period.
func (p *process) stop(grace time.Duration) {
	if p.alive() {
		_, _ = p.request(request{Op: "stop"}, grace)
		_ = p.stdin.Close()
		select {
		case <-p.done:
		case <-time.After(grace):
			p.kill()
		}
	}
	<-p.done
}

// tail keeps the last bytes the helper wrote to stderr.
type tail struct {
	mu    sync.Mutex
	limit int
	data  []byte
}

func newTail(limit int) *tail { return &tail{limit: limit} }

func (t *tail) Write(b []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.data = append(t.data, b...)
	if len(t.data) > t.limit {
		t.data = append([]byte(nil), t.data[len(t.data)-t.limit:]...)
	}
	return len(b), nil
}

func (t *tail) lastLine() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	end := len(t.data)
	for end > 0 && (t.data[end-1] == '\n' || t.data[end-1] == '\r') {
		end--
	}
	start := end
	for start > 0 && t.data[start-1] != '\n' {
		start--
	}
	return string(t.data[start:end])
}
