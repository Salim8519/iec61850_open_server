package desktop

import (
	"fmt"
	"os"
)

// Headless is windowed.headless: true when nobody can read stdout or stderr, as for the
// double-clicked windowed EXE. Like Python's sys.stdout is None, a stream counts as
// missing when its handle is absent or of unknown type (see stdUsable).
func Headless() bool {
	return !stdUsable(os.Stdout) || !stdUsable(os.Stderr)
}

// TellUser is windowed.tell_user: print text (stderr for errors, stdout otherwise) when a
// terminal exists, otherwise show a message box titled "ScadaBench" with the error
// (MB_ICONERROR) or information (MB_ICONINFORMATION) icon. The box blocks until closed.
func TellUser(text string, isError bool) {
	if !Headless() {
		stream := os.Stdout
		if isError {
			stream = os.Stderr
		}
		_, _ = fmt.Fprint(stream, text+lineEnd)
		return
	}
	messageBox(text, isError)
}
