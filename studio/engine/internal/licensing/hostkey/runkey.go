package hostkey

import "sync"

// RunKey is the one value that starts the host helper at logon: HKCU\Software\Microsoft\Windows\
// CurrentVersion\Run, value RunValueName. It exists only while "Let my virtual machines use this
// license" is on, so it is also the switch the helper watches.
type RunKey interface {
	// Get returns the command line, or false when the value is missing.
	Get() (string, bool)
	Set(command string) error
	Delete() error
}

// MemoryRunKey is the fake of the Run key for tests.
type MemoryRunKey struct {
	mu    sync.Mutex
	value string
	set   bool
}

// Get returns the stored command line.
func (m *MemoryRunKey) Get() (string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.value, m.set
}

// Set stores the command line.
func (m *MemoryRunKey) Set(command string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value, m.set = command, true
	return nil
}

// Delete removes the value.
func (m *MemoryRunKey) Delete() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value, m.set = "", false
	return nil
}
