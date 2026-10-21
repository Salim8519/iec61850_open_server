//go:build !windows

package desktop

import (
	"os"
	"syscall"
)

// newProcessGroup adds nothing outside Windows (ScadaBench is a Windows application).
func newProcessGroup() *syscall.SysProcAttr { return nil }

// openDefault is not provided outside Windows.
func openDefault(url string) {}

// deleteFile removes the file (close enough to os.unlink for a build-only stub).
func deleteFile(path string) error { return os.Remove(path) }
