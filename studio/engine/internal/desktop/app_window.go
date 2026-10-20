package desktop

import "sync"

// AppWindow is app_window.AppWindow, the packaged app's single window: open it, show it
// again when the user starts ScadaBench a second time, and stop ScadaBench when it closes.
// Packaged launches only; development never opens windows (ADR-0012).
type AppWindow struct {
	URL     string // the address the window opens (the caller adds "&window=app" when FindBrowser finds one)
	Profile string // the window's own browser profile (ChooseProfile)

	onClosed func()
	log      *Logger
	sys      windowSystem
	mu       sync.Mutex
	process  process
}

// NewAppWindow chooses the window's browser profile in dataDir. onClosed runs at most
// once, on the watcher goroutine, when the window is really gone; log may be nil.
func NewAppWindow(url, dataDir string, onClosed func(), log *Logger) (*AppWindow, error) {
	profile, err := ChooseProfile(dataDir)
	if err != nil {
		return nil, err
	}
	return newAppWindow(url, profile, onClosed, log, realSystem()), nil
}

func newAppWindow(url, profile string, onClosed func(), log *Logger, sys windowSystem) *AppWindow {
	var once sync.Once
	return &AppWindow{URL: url, Profile: profile, log: log, sys: sys,
		onClosed: func() { once.Do(onClosed) }}
}

// Open starts Edge/Chrome as a chromeless --app window on the profile (new process
// group, null stdio) and watches it; with no browser, or when it cannot start, the URL
// opens in the default browser instead and nothing is watched (PID stays 0).
func (w *AppWindow) Open() *AppWindow {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.process = w.start()
	if w.process != nil {
		watchWindow(w.sys, w.process, w.Profile, w.log, w.onClosed)
	}
	return w
}

// PID is the started browser's process id (for launcher.json window_pid), or 0 when the
// default browser was used.
func (w *AppWindow) PID() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.process == nil {
		return 0
	}
	return w.process.PID()
}

// Show is AppWindow.show, called when the user starts ScadaBench again: focus the running
// window; do nothing when it was closed (ScadaBench is already stopping); reopen the URL
// in the default browser when no app window was started; otherwise start the browser on
// the profile again, which opens a window in the browser process that holds it.
func (w *AppWindow) Show() {
	w.mu.Lock()
	defer w.mu.Unlock()
	started := w.process
	if started != nil && !started.Exited() && w.sys.focus(started.PID()) {
		return
	}
	if started != nil && started.Exited() && !w.sys.inUse(w.Profile) {
		return
	}
	if started == nil {
		w.sys.openDefault(w.URL)
		return
	}
	w.start()
}

// start is app_window.open_app_window: the started browser, or nil after falling back to
// the default browser.
func (w *AppWindow) start() process {
	browser := w.sys.findBrowser()
	if browser == "" {
		w.sys.openDefault(w.URL)
		return nil
	}
	started, err := w.sys.start(AppWindowCommand(browser, w.URL, w.Profile))
	if err != nil {
		w.sys.openDefault(w.URL)
		return nil
	}
	return started
}
