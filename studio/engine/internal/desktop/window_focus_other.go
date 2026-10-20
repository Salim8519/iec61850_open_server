//go:build !windows

package desktop

// FocusProcessWindow cannot focus windows outside Windows and reports false.
func FocusProcessWindow(pid int) bool { return false }

// AllowForeground does nothing outside Windows.
func AllowForeground(pid int) {}
