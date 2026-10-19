package worker

import (
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW, the flag WorkerProcess passes on Windows.
const createNoWindow = 0x08000000

func hideWindow(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
