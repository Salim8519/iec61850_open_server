//go:build windows

package xsd

import (
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW, as Python's WorkerProcess starts native helpers.
const createNoWindow = 0x08000000

func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
