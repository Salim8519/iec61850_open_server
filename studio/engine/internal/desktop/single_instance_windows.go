//go:build windows

package desktop

import (
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	procCreateEventW             = kernel32.NewProc("CreateEventW")
	procOpenEventW               = kernel32.NewProc("OpenEventW")
	procSetEvent                 = kernel32.NewProc("SetEvent")
	procWaitForSingleObject      = kernel32.NewProc("WaitForSingleObject")
	procAllowSetForegroundWindow = user32.NewProc("AllowSetForegroundWindow")
)

const (
	eventModifyState = 0x0002
	infinite         = 0xFFFFFFFF
	waitObject0      = 0
	asfwAny          = 0xFFFFFFFF // ASFW_ANY (DWORD)-1
	// listenerCloseWait bounds how long close waits for a running onShow to return.
	listenerCloseWait = 2 * time.Second
)

// RequestShow is single_instance.request_show: open the running instance's named event
// and signal it. Because this process was just started by the user it first passes its
// right to take the foreground on (AllowSetForegroundWindow(ASFW_ANY)). True when the
// running instance was asked; false when nobody listens on dir.
func RequestShow(dir string) bool {
	name, err := syscall.UTF16PtrFromString(ShowEventName(dir))
	if err != nil {
		return false
	}
	handle, _, _ := procOpenEventW.Call(eventModifyState, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(handle))
	_, _, _ = procAllowSetForegroundWindow.Call(asfwAny)
	signalled, _, _ := procSetEvent.Call(handle)
	return signalled != 0
}

// StartShowListener is single_instance.ShowListener.start: create the auto-reset event
// for dir and call onShow each time another launch signals it. A panic in onShow is
// logged and the listener keeps running. When the event cannot be created nothing
// listens. The returned close (ShowListener.close) stops the listener; it is idempotent
// and waits up to 2 s for a running onShow to return.
func StartShowListener(dir string, onShow func(), log *Logger) (close func()) {
	name, err := syscall.UTF16PtrFromString(ShowEventName(dir))
	if err != nil {
		return func() {}
	}
	handle, _, _ := procCreateEventW.Call(0, 0, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return func() {}
	}
	listener := &showListener{handle: handle, done: make(chan struct{})}
	go listener.run(onShow, log)
	return listener.close
}

type showListener struct {
	mu      sync.Mutex
	handle  uintptr // zero once the run goroutine closed it
	stopped bool
	once    sync.Once
	done    chan struct{}
}

func (l *showListener) run(onShow func(), log *Logger) {
	defer close(l.done)
	defer l.release()
	for {
		result, _, _ := procWaitForSingleObject.Call(l.handle, infinite)
		if result != waitObject0 || l.isStopped() {
			return
		}
		runShow(onShow, log)
	}
}

func (l *showListener) isStopped() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stopped
}

// release closes the event handle from the goroutine that waits on it, so close never
// signals a handle that was already closed (and possibly reused).
func (l *showListener) release() {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = syscall.CloseHandle(syscall.Handle(l.handle))
	l.handle = 0
}

func (l *showListener) close() {
	l.once.Do(func() {
		l.mu.Lock()
		l.stopped = true
		if l.handle != 0 {
			_, _, _ = procSetEvent.Call(l.handle)
		}
		l.mu.Unlock()
		select {
		case <-l.done:
		case <-time.After(listenerCloseWait):
		}
	})
}
