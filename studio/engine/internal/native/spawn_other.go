//go:build !windows

package native

import (
	"errors"
	"os/exec"
)

func prepareCommand(command *exec.Cmd) {}

// adopt has no Job Object outside Windows; the product ships for Windows only.
func adopt(pid int) error { return errors.New("job objects need Windows") }
