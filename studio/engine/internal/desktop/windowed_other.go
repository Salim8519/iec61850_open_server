//go:build !windows

package desktop

import "os"

// lineEnd is what Python's text streams write for "\n" outside Windows.
const lineEnd = "\n"

// stdUsable reports whether a standard stream refers to an open file.
func stdUsable(file *os.File) bool {
	if file == nil {
		return false
	}
	_, err := file.Stat()
	return err == nil
}

// AttachParentTerminal does nothing outside Windows (there is no windowed EXE).
func AttachParentTerminal() {}

// messageBox has no equivalent outside Windows; Python showed nothing there either.
func messageBox(text string, isError bool) {}
