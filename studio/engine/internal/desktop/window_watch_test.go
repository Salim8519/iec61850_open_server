package desktop

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeWorld is a windowSystem with a manual clock and a scripted profile lock.
type fakeWorld struct {
	mu       sync.Mutex
	clock    time.Time
	slept    time.Duration
	inUse    func(call int) bool
	calls    int
	browser  string
	startErr error
	started  []*fakeProcess
	defaults []string
	focused  []int
	focusOK  bool
}

func (f *fakeWorld) system() windowSystem {
	return windowSystem{
		findBrowser: func() string { return f.browser },
		start: func(command []string) (process, error) {
			f.mu.Lock()
			defer f.mu.Unlock()
			if f.startErr != nil {
				return nil, f.startErr
			}
			p := &fakeProcess{pid: 100 + len(f.started), exit: make(chan struct{})}
			f.started = append(f.started, p)
			return p, nil
		},
		openDefault: func(url string) { f.mu.Lock(); f.defaults = append(f.defaults, url); f.mu.Unlock() },
		focus:       func(pid int) bool { f.focused = append(f.focused, pid); return f.focusOK },
		inUse: func(string) bool {
			f.mu.Lock()
			defer f.mu.Unlock()
			f.calls++
			return f.inUse(f.calls)
		},
		now: func() time.Time { f.mu.Lock(); defer f.mu.Unlock(); return f.clock },
		sleep: func(d time.Duration) {
			f.mu.Lock()
			f.clock, f.slept = f.clock.Add(d), f.slept+d
			f.mu.Unlock()
		},
	}
}

type fakeProcess struct {
	pid  int
	exit chan struct{}
}

func (p *fakeProcess) PID() int { return p.pid }
func (p *fakeProcess) Wait()    { <-p.exit }
func (p *fakeProcess) Exited() bool {
	select {
	case <-p.exit:
		return true
	default:
		return false
	}
}

func never(int) bool  { return false }
func always(int) bool { return true }

func TestHandoffQuickExitWaitsGraceThenGivesUp(t *testing.T) {
	world := &fakeWorld{clock: time.Unix(1000, 0), inUse: never}
	sys := world.system()
	if waitForHandoff(sys, "p", world.clock.Add(-2*time.Second)) {
		t.Fatal("no process took the profile, so no hand-off")
	}
	if world.slept != HandoffGrace || world.calls != 11 {
		t.Fatalf("slept %v with %d checks, want %v and 11", world.slept, world.calls, HandoffGrace)
	}
}

func TestHandoffQuickExitSeesBrokerTakeProfile(t *testing.T) {
	world := &fakeWorld{clock: time.Unix(1000, 0), inUse: func(call int) bool { return call >= 3 }}
	if !waitForHandoff(world.system(), "p", world.clock) {
		t.Fatal("the profile was taken during the grace period")
	}
	if world.slept != 2*PollInterval {
		t.Fatalf("slept %v, want %v", world.slept, 2*PollInterval)
	}
}

func TestHandoffLateExitChecksOnce(t *testing.T) {
	world := &fakeWorld{clock: time.Unix(1000, 0), inUse: never}
	if waitForHandoff(world.system(), "p", world.clock.Add(-QuickExit-time.Second)) {
		t.Fatal("late exit with a free profile is no hand-off")
	}
	if world.slept != 0 || world.calls != 1 {
		t.Fatalf("late exit slept %v with %d checks", world.slept, world.calls)
	}
}

func openWatched(t *testing.T, world *fakeWorld) (*AppWindow, chan int) {
	t.Helper()
	closed := make(chan int, 4)
	count := 0
	window := newAppWindow("http://x/#s", "prof", func() { count++; closed <- count }, nil, world.system()).Open()
	return window, closed
}

func TestWindowClosesAfterHandedOffProcessReleasesProfile(t *testing.T) {
	// Checks 1-2 free, 3-6 held by a broker process, then released.
	world := &fakeWorld{clock: time.Unix(1000, 0), browser: "edge.exe",
		inUse: func(call int) bool { return call >= 3 && call <= 6 }}
	window, closed := openWatched(t, world)
	if window.PID() != 100 {
		t.Fatalf("PID %d", window.PID())
	}
	close(world.started[0].exit)
	select {
	case n := <-closed:
		if n != 1 {
			t.Fatalf("onClosed count %d", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("onClosed never ran")
	}
	if world.calls != 7 {
		t.Fatalf("profile checked %d times, want 7", world.calls)
	}
	window.onClosed()
	if len(closed) != 0 {
		t.Fatal("onClosed ran twice")
	}
}

func TestOpenFallsBackToDefaultBrowser(t *testing.T) {
	world := &fakeWorld{inUse: never}
	window, _ := openWatched(t, world)
	if window.PID() != 0 || len(world.defaults) != 1 {
		t.Fatalf("no browser: pid %d defaults %v", window.PID(), world.defaults)
	}
	world = &fakeWorld{inUse: never, browser: "edge.exe", startErr: errors.New("denied")}
	window, _ = openWatched(t, world)
	if window.PID() != 0 || len(world.defaults) != 1 {
		t.Fatalf("start failed: pid %d defaults %v", window.PID(), world.defaults)
	}
	window.Show()
	if len(world.defaults) != 2 {
		t.Fatal("Show without an app window reopens the default browser")
	}
}

func TestShowFocusesRunningWindow(t *testing.T) {
	world := &fakeWorld{inUse: always, browser: "edge.exe", focusOK: true}
	window, _ := openWatched(t, world)
	window.Show()
	if len(world.focused) != 1 || world.focused[0] != 100 || len(world.started) != 1 {
		t.Fatalf("focused %v started %d", world.focused, len(world.started))
	}
	world.focusOK = false
	window.Show()
	if len(world.started) != 2 {
		t.Fatal("an unfocusable running window is opened again on the profile")
	}
}

func TestShowAfterCloseDoesNothing(t *testing.T) {
	world := &fakeWorld{inUse: never, browser: "edge.exe"}
	window, closed := openWatched(t, world)
	world.mu.Lock()
	world.clock = world.clock.Add(time.Hour) // late exit: the watcher checks the profile once
	world.mu.Unlock()
	close(world.started[0].exit)
	select {
	case <-closed:
	case <-time.After(5 * time.Second):
		t.Fatal("onClosed never ran")
	}
	window.Show()
	if len(world.started) != 1 || len(world.focused) != 0 || len(world.defaults) != 0 {
		t.Fatalf("closed window: started %d focused %v defaults %v", len(world.started), world.focused, world.defaults)
	}
	world.mu.Lock()
	world.inUse = always
	world.mu.Unlock()
	window.Show()
	if len(world.started) != 2 {
		t.Fatal("an exited launcher whose profile is still held reopens on that profile")
	}
}
