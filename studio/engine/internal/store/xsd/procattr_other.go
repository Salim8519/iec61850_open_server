//go:build !windows

package xsd

import "os/exec"

func hideWindow(*exec.Cmd) {}
