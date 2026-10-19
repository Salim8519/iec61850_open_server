package xsd

import (
	"errors"
	"os"
	"sync"
	"time"
)

// helloTimeout bounds the identification of a freshly started helper.
const helloTimeout = 15 * time.Second

// stopGrace is how long Close waits for the helper to exit before killing it.
const stopGrace = 3 * time.Second

// client owns the studio_xsd process: started lazily, identified with hello before use,
// one validation at a time (Python validated under a global lock), killed on timeout and
// restarted by the next request after a crash.
type client struct {
	path          string
	expectedBuild string
	timeout       time.Duration
	work          *workFiles
	turn          chan struct{} // one validation at a time
	mu            sync.Mutex    // guards proc and closed
	proc          *process
	closed        bool
	starts        int // helper processes started, for tests and diagnostics
}

func newClient(path, expectedBuild string, timeout time.Duration, work *workFiles) *client {
	return &client{path: path, expectedBuild: expectedBuild, timeout: timeout, work: work,
		turn: make(chan struct{}, 1)}
}

// running returns a live, identified helper, starting one when needed.
func (c *client) running() (*process, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, errClosed()
	}
	if c.proc != nil && c.proc.alive() {
		return c.proc, nil
	}
	c.proc = nil
	if info, err := os.Stat(c.path); err != nil || info.IsDir() {
		return nil, errMissing()
	}
	proc, err := startProcess(c.path)
	if err != nil {
		return nil, errStart(err)
	}
	c.starts++
	hello, err := proc.request(request{Op: "hello"}, helloTimeout)
	if err == nil {
		err = checkHello(hello, c.expectedBuild)
	} else if errors.Is(err, errTimedOut) {
		err = errProtocol("no answer to hello")
	}
	if err != nil {
		proc.kill()
		<-proc.done
		return nil, err
	}
	c.proc = proc
	return proc, nil
}

// validate runs one document through the helper; the bytes travel as a work file
// because a 20 MB SCL does not fit a 16 MiB frame.
func (c *client) validate(source []byte, schemaPath string) (outcome, error) {
	c.turn <- struct{}{}
	defer func() { <-c.turn }()
	proc, err := c.running()
	if err != nil {
		return outcome{}, err
	}
	path, err := c.work.write(source)
	if err != nil {
		return outcome{}, err
	}
	defer c.work.remove(path)
	raw, err := proc.request(request{Op: "validate", SchemaPath: schemaPath, SourcePath: path}, c.timeout)
	if errors.Is(err, errTimedOut) {
		proc.kill() // libxml2 cannot be interrupted; the next request starts a new helper
		<-proc.done
		return outcome{}, errTimeout(c.timeout)
	}
	if err != nil {
		return outcome{}, err
	}
	return decodeOutcome(raw)
}

// close stops the helper; later requests fail with errClosed.
func (c *client) close() {
	c.mu.Lock()
	c.closed = true
	proc := c.proc
	c.proc = nil
	c.mu.Unlock()
	if proc != nil {
		proc.stop(stopGrace)
	}
}

func (c *client) startCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.starts
}
