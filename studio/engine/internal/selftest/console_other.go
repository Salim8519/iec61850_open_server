//go:build !windows

package selftest

import "os/exec"

// hideConsole has nothing to hide outside Windows.
func hideConsole(*exec.Cmd) {}
