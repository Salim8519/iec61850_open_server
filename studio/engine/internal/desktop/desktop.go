// Package desktop is the packaged ScadaBench's window behaviour (ADR-0083), ported from
// the Python launcher: the chromeless Edge/Chrome app window and its stop-on-close rule
// (app_window.py), one instance per data folder with a "show yourself" event
// (single_instance.py), window focus (window_focus.py), message boxes, terminal attach and
// the launcher log for the windowed EXE (windowed.py), the SubstationStudio folder move
// (legacy_folder.py) and the second-launch hand-off (entrypoint.show_running_instance).
//
// Windows APIs are called through syscall.NewLazyDLL; other systems get inert stubs so
// the package builds and vets everywhere. Nothing runs at import time.
package desktop

// Title is the caption of every message box ScadaBench shows.
const Title = "ScadaBench"

// Python logger names, kept so Go and Python launcher logs read the same.
const (
	appWindowLogger      = "studio_app.launcher.app_window"
	singleInstanceLogger = "studio_app.launcher.single_instance"
)

// AlreadyRunningText is printed to a terminal when a second launch focused the first one.
const AlreadyRunningText = "ScadaBench is already running; its window was brought to the front."
