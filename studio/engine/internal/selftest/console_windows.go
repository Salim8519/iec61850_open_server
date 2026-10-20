//go:build windows

package selftest

import (
	"os/exec"
	"syscall"
)

// createNoWindow is subprocess.CREATE_NO_WINDOW, as self_test.read_native starts the probe.
const createNoWindow = 0x08000000

func hideConsole(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
