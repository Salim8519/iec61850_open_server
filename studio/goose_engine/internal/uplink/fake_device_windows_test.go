package uplink

import (
	"errors"
	"sync"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	procResetEvent     = kernel32.NewProc("ResetEvent")
	procQueryCounter   = kernel32.NewProc("QueryPerformanceCounter")
	procQueryFrequency = kernel32.NewProc("QueryPerformanceFrequency")
)

// fakeDevice behaves like an Npcap handle in non-blocking mode: a manual-reset
// receive event that every read clears before copying frames, exactly as the Npcap
// driver does (NPF_Read), and a Send that records what was handed to it.
type fakeDevice struct {
	event   syscall.Handle
	mu      sync.Mutex
	inbound [][]byte
	sent    [][]byte
	sentAt  []int64 // QueryPerformanceCounter ticks at hand-off
	readErr error
	sendErr error
}

func newFakeDevice(t *testing.T) *fakeDevice {
	t.Helper()
	event, err := createEvent(true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { syscall.CloseHandle(event) })
	return &fakeDevice{event: event}
}

// opener returns the device once; later reopen attempts fail like a missing adapter.
func (d *fakeDevice) opener() opener {
	var once sync.Once
	return func(string) (device, error) {
		var opened device
		once.Do(func() { opened = d })
		if opened == nil {
			return nil, errors.New("adapter gone")
		}
		return opened, nil
	}
}

// arrive queues frames as if heard on the wire and signals the receive event.
func (d *fakeDevice) arrive(frames ...[]byte) {
	d.mu.Lock()
	d.inbound = append(d.inbound, frames...)
	d.mu.Unlock()
	procSetEvent.Call(uintptr(d.event))
}

func (d *fakeDevice) fail(err error) {
	d.mu.Lock()
	d.readErr = err
	d.mu.Unlock()
	procSetEvent.Call(uintptr(d.event))
}

func (d *fakeDevice) Next() ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	procResetEvent.Call(uintptr(d.event))
	if d.readErr != nil {
		return nil, d.readErr
	}
	if len(d.inbound) == 0 {
		return nil, nil
	}
	frame := d.inbound[0]
	d.inbound = d.inbound[1:]
	return frame, nil
}

func (d *fakeDevice) Send(frame []byte) error {
	at := qpc()
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.sendErr != nil {
		return d.sendErr
	}
	d.sent = append(d.sent, frame)
	d.sentAt = append(d.sentAt, at)
	return nil
}

func (d *fakeDevice) Event() uintptr { return uintptr(d.event) }
func (d *fakeDevice) Close()         {}

func (d *fakeDevice) sentCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.sent)
}

// qpc is QueryPerformanceCounter; Go's clock on Windows can be as coarse as the
// system timer tick, too coarse for microsecond latencies.
func qpc() int64 {
	var ticks int64
	procQueryCounter.Call(uintptr(unsafe.Pointer(&ticks)))
	return ticks
}

func qpcFrequency() int64 {
	var hz int64
	procQueryFrequency.Call(uintptr(unsafe.Pointer(&hz)))
	return hz
}

// recorder collects onFrame and onState calls.
type recorder struct {
	mu     sync.Mutex
	frames [][]byte
	states []string
	notify chan string
}

func newRecorder() *recorder { return &recorder{notify: make(chan string, 64)} }

func (r *recorder) frame(b []byte) {
	r.mu.Lock()
	r.frames = append(r.frames, b)
	r.mu.Unlock()
}

func (r *recorder) state(state, _ string) {
	r.mu.Lock()
	r.states = append(r.states, state)
	r.mu.Unlock()
	select {
	case r.notify <- state:
	default:
	}
}

func (r *recorder) frameCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.frames)
}

func (r *recorder) await(t *testing.T, state string) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case got := <-r.notify:
			if got == state {
				return
			}
		case <-deadline:
			t.Fatalf("no %q state", state)
		}
	}
}

// eventually polls cond for up to two seconds.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}
