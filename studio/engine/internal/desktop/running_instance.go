package desktop

import (
	"fmt"
	"os"

	"studio-engine/internal/launcher"
)

// ShowRunningInstance is entrypoint.show_running_instance, for a launch that lost the
// data-folder lease to another ScadaBench: this freshly started process may take the
// foreground, so it focuses the window_pid recorded in launcher.json itself, otherwise
// asks the running instance to show or reopen its window (RequestShow). On success it
// prints AlreadyRunningText to stderr when a terminal exists and returns 0; otherwise it
// tells the user leaseErr (message box when headless) and returns 1.
func ShowRunningInstance(dataDir string, leaseErr error) int {
	pid := launcher.ReadWindowPID(dataDir)
	if (pid != 0 && FocusProcessWindow(pid)) || RequestShow(dataDir) {
		if !Headless() {
			_, _ = fmt.Fprint(os.Stderr, AlreadyRunningText+lineEnd)
		}
		return 0
	}
	text := ""
	if leaseErr != nil {
		text = leaseErr.Error()
	}
	TellUser(text, true)
	return 1
}
