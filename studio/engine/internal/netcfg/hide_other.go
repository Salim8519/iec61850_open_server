//go:build !windows

package netcfg

import "os/exec"

// hideWindow does nothing off Windows (creationflags are 0 there in Python).
func hideWindow(process *exec.Cmd) {}
