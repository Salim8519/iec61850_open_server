// Command scadabench_portable is ScadaBench-Portable-<version>.exe (ADR-0083, ADR-0063): one
// exe that runs ScadaBench directly. It holds the exact installed ScadaBench folder, copies it
// once to ScadaBench-data/app/<content hash> beside itself and runs that ScadaBench.exe with
// ScadaBench-data as its data folder, so projects, logs and workers stay beside the exe and
// the workers keep one stable path per build (Windows firewall rules name the program path).
package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"studio-engine/internal/desktop"
)

func main() { os.Exit(run(os.Args[1:])) }

func run(args []string) int {
	desktop.AttachParentTerminal()
	executable, err := os.Executable()
	if err != nil {
		return fail(err)
	}
	data := filepath.Join(filepath.Dir(executable), dataFolder)
	folder, err := stage(payload, data)
	if err != nil {
		return fail(err)
	}
	command := exec.Command(filepath.Join(folder, "ScadaBench.exe"), engineArguments(args, data)...)
	command.Dir = folder
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Start(); err != nil {
		return fail(err)
	}
	// This exe was started by the user; hand the foreground right to ScadaBench, which may
	// have to bring an already running window to the front.
	desktop.AllowForeground(command.Process.Pid)
	err = command.Wait()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	if err != nil {
		return fail(err)
	}
	return 0
}

// engineArguments passes the user's arguments and adds --data-dir <data> unless they name a
// data folder or start a helper mode that takes no options.
func engineArguments(args []string, data string) []string {
	if len(args) > 0 && (args[0] == "--network-helper" || args[0] == "--fold-journal") {
		return args
	}
	for _, argument := range args {
		if argument == "--data-dir" || strings.HasPrefix(argument, "--data-dir=") {
			return args
		}
	}
	return append([]string{"--data-dir", data}, args...)
}

func fail(err error) int {
	desktop.TellUser("ScadaBench could not start: "+err.Error(), true)
	return 1
}
