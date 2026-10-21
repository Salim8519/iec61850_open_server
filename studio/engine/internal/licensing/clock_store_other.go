//go:build !windows

package licensing

// WindowsRegistry has no registry outside Windows: reads find nothing and writes are
// ignored, as clock_store.WindowsRegistry does when winreg cannot be imported.
type WindowsRegistry struct{ Key string }

// Get finds nothing.
func (WindowsRegistry) Get(string) (string, bool) { return "", false }

// Set does nothing.
func (WindowsRegistry) Set(string, string) {}
