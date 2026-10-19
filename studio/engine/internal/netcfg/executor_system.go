package netcfg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
)

// SystemExecutor runs programs for real, without a console window. Only production code uses
// it; tests always pass a recording fake.
type SystemExecutor struct{}

// Run starts the program, waits up to the timeout and decodes its output.
func (SystemExecutor) Run(command Command) (Output, error) {
	ctx, cancel := context.WithTimeout(context.Background(), command.Timeout)
	defer cancel()
	process := exec.CommandContext(ctx, command.Argv[0], command.Argv[1:]...)
	hideWindow(process)
	var stdout, stderr bytes.Buffer
	process.Stdout, process.Stderr = &stdout, &stderr
	err := process.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return Output{}, fmt.Errorf("Command '%s' timed out after %g seconds", command.Argv[0], command.Timeout.Seconds())
	}
	output := Output{Stdout: decodeConsole(stdout.Bytes()), Stderr: decodeConsole(stderr.Bytes())}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		output.ExitCode = exit.ExitCode()
		return output, nil
	}
	return output, err
}
