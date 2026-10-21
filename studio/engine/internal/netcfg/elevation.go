package netcfg

import (
	"errors"
	"strings"
	"time"
)

// HelperFlag is the argv switch that makes the app exe run the elevated network helper.
const HelperFlag = "--network-helper"

// HelperTimeout is how long the app waits for the elevated helper (elevation.run_elevated).
const HelperTimeout = 120 * time.Second

// ErrElevationCancelled is elevation.ElevationCancelled: the user declined the Windows
// administrator prompt.
var ErrElevationCancelled = errors.New("Windows administrator permission was not given, so nothing was changed")

// Elevator starts a program with the Windows administrator (UAC) prompt and waits for it. The
// real one is UACElevator; tests use a fake that never elevates.
type Elevator interface {
	Elevate(executable string, arguments []string) (exitCode uint32, err error)
}

// ElevatorFunc adapts a function to Elevator.
type ElevatorFunc func(executable string, arguments []string) (uint32, error)

// Elevate calls the function.
func (f ElevatorFunc) Elevate(executable string, arguments []string) (uint32, error) {
	return f(executable, arguments)
}

// HelperCommand is elevation.helper_command for the packaged app: the same exe with
// --network-helper <request> <sha256 hex> <worker program>.
func HelperCommand(executable, requestPath, digest, program string) (string, []string) {
	return executable, []string{HelperFlag, requestPath, digest, program}
}

// ListToCmdline is subprocess.list2cmdline: the MS C runtime quoting rules.
func ListToCmdline(arguments []string) string {
	var result strings.Builder
	for index, argument := range arguments {
		if index > 0 {
			result.WriteByte(' ')
		}
		quote := argument == "" || strings.ContainsAny(argument, " \t")
		if quote {
			result.WriteByte('"')
		}
		backslashes := 0
		for _, c := range argument {
			switch c {
			case '\\':
				backslashes++
			case '"':
				result.WriteString(strings.Repeat(`\`, backslashes*2))
				backslashes = 0
				result.WriteString(`\"`)
			default:
				result.WriteString(strings.Repeat(`\`, backslashes))
				backslashes = 0
				result.WriteRune(c)
			}
		}
		result.WriteString(strings.Repeat(`\`, backslashes))
		if quote {
			result.WriteString(strings.Repeat(`\`, backslashes))
			result.WriteByte('"')
		}
	}
	return result.String()
}
