//go:build windows

package netcfg

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// hideWindow starts the process with CREATE_NO_WINDOW, as Python's subprocess calls do.
func hideWindow(process *exec.Cmd) {
	process.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
