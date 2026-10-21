package netcfg

import (
	"time"
	"unicode/utf8"

	"studio-engine/internal/apperr"
)

// Command is one program start: the exact argv and how long it may run.
type Command struct {
	Argv    []string
	Timeout time.Duration
}

// Output is what a finished program wrote, decoded to text like subprocess text=True.
type Output struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Executor starts programs. The real one (SystemExecutor) runs them hidden; tests record the
// commands instead. An error means the program could not run or timed out.
type Executor interface {
	Run(command Command) (Output, error)
}

// ExecutorFunc adapts a function to Executor.
type ExecutorFunc func(command Command) (Output, error)

// Run calls the function.
func (f ExecutorFunc) Run(command Command) (Output, error) { return f(command) }

// PowerShell is the argv every networking module uses for a script.
func PowerShell(script string) []string {
	return []string{"powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script}
}

// shellQuery runs a read-only PowerShell script like the modules' _powershell helpers: a
// non-zero exit becomes RuntimeError(prefix + last stderr line or "unknown error").
func shellQuery(run Executor, script string, timeout time.Duration, prefix string) (string, error) {
	output, err := run.Run(Command{Argv: PowerShell(script), Timeout: timeout})
	if err != nil {
		return "", err
	}
	if output.ExitCode != 0 {
		return "", apperr.RuntimeError(prefix + lastLine(output.Stderr, "unknown error"))
	}
	return output.Stdout, nil
}

// decodeConsole turns program output into text: UTF-8 when valid, else one character per
// byte (Python decodes with the ANSI code page; both agree on ASCII).
func decodeConsole(data []byte) string {
	if utf8.Valid(data) {
		return string(data)
	}
	runes := make([]rune, len(data))
	for index, value := range data {
		runes[index] = rune(value)
	}
	return string(runes)
}
