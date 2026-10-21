package main

import (
	"fmt"
	"os"

	"studio-engine/internal/desktop"
	"studio-engine/internal/launcher"
)

// startWindow is entrypoint.start_window: the packaged app opens its window (unless
// --no-browser) and every non-development launch answers "show yourself" requests from a
// second start. closed is closed when the window is really gone, which stops ScadaBench.
// The returned function stops the show listener.
func startWindow(options launcher.Options, data, url, target string, log *desktop.Logger,
	closed chan struct{}) func() {
	var window *desktop.AppWindow
	if packaged == "1" && !options.NoBrowser {
		address := target
		if desktop.FindBrowser(os.Getenv) != "" {
			address += "&window=app"
		}
		created, err := desktop.NewAppWindow(address, data, func() { close(closed) }, log)
		if err != nil {
			log.Error("studio_app.launcher.app_window", "Could not prepare the app window: %v", err)
		} else {
			window = created.Open()
			if pid := window.PID(); pid != 0 {
				_ = launcher.WriteRecord(data, url, pid)
			}
		}
	}
	show := func() {}
	if window != nil {
		show = window.Show
	}
	return desktop.StartShowListener(data, show, log)
}

// startFailed reports a start failure: in the terminal when there is one, otherwise in the
// launcher log and a message box (entrypoint.main's headless branch). Exit code 1.
func startFailed(log *desktop.Logger, data string, err error) int {
	if !desktop.Headless() {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	log.Error("studio_app.launcher.entrypoint", "ScadaBench could not start: %v", err)
	desktop.TellUser(fmt.Sprintf("ScadaBench could not start: %v\n\nDetails are in the logs folder of %s.", err, data), true)
	return 1
}

// engineLog sends the engine's diagnostics to the launcher log file when nobody can read
// the terminal (double-clicked exe); with a terminal the engine keeps its own output.
func engineLog(log *desktop.Logger) func(format string, args ...any) {
	if !desktop.Headless() {
		return nil
	}
	return func(format string, args ...any) { log.Warning("studio_app", format, args...) }
}
