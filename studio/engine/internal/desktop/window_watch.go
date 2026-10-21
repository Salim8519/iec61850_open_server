package desktop

import (
	"os"
	"time"
)

// Timing of app_window.py. Edge/Chrome may relaunch itself or pass the window to a
// broker process right after start.
const (
	QuickExit    = 15 * time.Second
	HandoffGrace = 5 * time.Second
	PollInterval = 500 * time.Millisecond
)

// windowSystem is everything AppWindow does to the outside world; tests replace it.
type windowSystem struct {
	findBrowser func() string
	start       func(command []string) (process, error)
	openDefault func(url string)
	focus       func(pid int) bool
	inUse       func(profile string) bool
	now         func() time.Time
	sleep       func(time.Duration)
}

func realSystem() windowSystem {
	return windowSystem{
		findBrowser: func() string { return FindBrowser(os.Getenv) },
		start:       startBrowser,
		openDefault: openDefault,
		focus:       FocusProcessWindow,
		inUse:       ProfileInUse,
		now:         time.Now,
		sleep:       time.Sleep,
	}
}

// waitForHandoff is app_window.wait_for_handoff: a browser that exits within QuickExit
// of its start usually handed the window on, so wait up to HandoffGrace (polling every
// PollInterval) for another process to hold the profile. A late exit checks once.
func waitForHandoff(sys windowSystem, profile string, started time.Time) bool {
	if sys.now().Sub(started) > QuickExit {
		return sys.inUse(profile)
	}
	deadline := sys.now().Add(HandoffGrace)
	for sys.now().Before(deadline) {
		if sys.inUse(profile) {
			return true
		}
		sys.sleep(PollInterval)
	}
	return sys.inUse(profile)
}

// watchWindow is app_window.watch_window: when the started browser exits, and no other
// browser process took over its profile (or that process released it too), call
// onClosed. The profile belongs to this window alone.
func watchWindow(sys windowSystem, started process, profile string, log *Logger, onClosed func()) {
	startedAt := sys.now()
	go func() {
		started.Wait()
		if waitForHandoff(sys, profile, startedAt) {
			log.Warning(appWindowLogger, "App window was handed to another browser process")
			for sys.inUse(profile) {
				sys.sleep(PollInterval)
			}
		}
		onClosed()
	}()
}
