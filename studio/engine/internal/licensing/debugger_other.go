//go:build !windows

package licensing

// DebuggerAttached is false outside Windows, as in integrity.py.
func DebuggerAttached() bool { return false }
