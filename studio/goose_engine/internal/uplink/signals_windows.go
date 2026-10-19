package uplink

import (
	"errors"
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procCreateEventW           = kernel32.NewProc("CreateEventW")
	procSetEvent               = kernel32.NewProc("SetEvent")
	procWaitForMultipleObjects = kernel32.NewProc("WaitForMultipleObjects")
)

const (
	infinite   = 0xFFFFFFFF
	waitFailed = 0xFFFFFFFF
)

// signals are the two Windows events the adapter thread waits on besides Npcap's
// receive event: stop (manual reset, stays set) and send (auto reset).
type signals struct {
	mu     sync.RWMutex // kick vs release: never signal a closed (maybe reused) handle
	closed bool
	stop   syscall.Handle
	send   syscall.Handle
}

func newSignals() (*signals, error) {
	stop, err := createEvent(true)
	if err != nil {
		return nil, err
	}
	send, err := createEvent(false)
	if err != nil {
		syscall.CloseHandle(stop)
		return nil, err
	}
	return &signals{stop: stop, send: send}, nil
}

func createEvent(manualReset bool) (syscall.Handle, error) {
	manual := uintptr(0)
	if manualReset {
		manual = 1
	}
	h, _, err := procCreateEventW.Call(0, manual, 0, 0)
	if h == 0 {
		return 0, err
	}
	return syscall.Handle(h), nil
}

// kick wakes the adapter thread to send queued frames.
func (s *signals) kick() { s.set(s.send) }

// halt wakes the adapter thread for good.
func (s *signals) halt() { s.set(s.stop) }

func (s *signals) set(event syscall.Handle) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.closed {
		procSetEvent.Call(uintptr(event))
	}
}

// wait blocks in the kernel until stop, send or the receive event (when not 0) is
// signalled. It never times out, so an idle adapter costs no wake-ups at all.
func (s *signals) wait(receive uintptr) error {
	handles := [3]syscall.Handle{s.stop, s.send, syscall.Handle(receive)}
	count := uintptr(3)
	if receive == 0 {
		count = 2
	}
	r, _, err := procWaitForMultipleObjects.Call(count, uintptr(unsafe.Pointer(&handles[0])), 0, infinite)
	if uint32(r) == waitFailed {
		if err == nil || err == syscall.Errno(0) {
			err = errors.New("unknown error")
		}
		return errors.New("waiting for the adapter failed: " + err.Error())
	}
	return nil
}

// release closes the events once the adapter thread has ended.
func (s *signals) release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.closed {
		s.closed = true
		syscall.CloseHandle(s.stop)
		syscall.CloseHandle(s.send)
	}
}
