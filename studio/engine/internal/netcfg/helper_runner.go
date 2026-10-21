package netcfg

import (
	"errors"
	"os"
	"time"

	"studio-engine/internal/pyjson"
)

// HelperCommandTimeout is how long one PowerShell change may take in the helper.
const HelperCommandTimeout = 60 * time.Second

// ChangeArgv is the argv helper/runner._run starts for one change command.
func ChangeArgv(command string) []string {
	return PowerShell("$ErrorActionPreference='Stop'; " + command + " | Out-Null")
}

// runChange is runner._run: ok when PowerShell exits 0; the message is the last line of
// stderr (or stdout when stderr is empty).
func runChange(executor Executor, command string) (bool, string, error) {
	output, err := executor.Run(Command{Argv: ChangeArgv(command), Timeout: HelperCommandTimeout})
	if err != nil {
		return false, "", err
	}
	text := output.Stderr
	if text == "" {
		text = output.Stdout
	}
	return output.ExitCode == 0, lastLine(text, ""), nil
}

// ExecuteRequest is runner.execute: every item of a validated request once, in order, with
// a result {key, ok, message[:500]} each.
func ExecuteRequest(request *pyjson.Object, executor Executor) ([]any, error) {
	items, _ := get(request, "items").([]any)
	results := []any{}
	for _, raw := range items {
		item, _ := raw.(*pyjson.Object)
		var command string
		var err error
		if get(request, "action") == "apply" {
			command, err = ApplyCommand(item)
		} else {
			command, err = RemoveCommand(item)
		}
		if err != nil {
			return nil, err
		}
		ok, message, err := runChange(executor, command)
		if err != nil {
			return nil, err
		}
		key, err := ItemKey(item)
		if err != nil {
			return nil, err
		}
		results = append(results, pyjson.NewObject().Set("key", key).Set("ok", ok).Set("message", cutRunes(message, 500)))
	}
	return results, nil
}

// HelperMain is runner.main inside the elevated helper. argv: request path, expected SHA-256,
// worker program path. It writes <request>.result.json and returns the exit code: 0 done,
// 2 refused, 1 when Python would have raised (no result file then).
func HelperMain(argv []string, executor Executor) int {
	if len(argv) < 3 {
		return 1
	}
	requestPath, expected, program := argv[0], argv[1], argv[2]
	payload, err := os.ReadFile(requestPath)
	if err != nil {
		return 1
	}
	outcome := pyjson.NewObject().Set("refused", false)
	request, err := ValidateRequest(payload, expected, program)
	var refused *HelperRefused
	switch {
	case errors.As(err, &refused):
		outcome.Set("refused", true).Set("reason", refused.Reason).Set("results", []any{})
	case err != nil:
		return 1
	default:
		results, err := ExecuteRequest(request, executor)
		if err != nil {
			return 1
		}
		outcome.Set("results", results)
	}
	data, err := pyjson.Dumps(outcome)
	if err != nil {
		return 1
	}
	if err := os.WriteFile(withSuffix(requestPath, ".result.json"), data, 0o666); err != nil {
		return 1
	}
	if refused != nil {
		return 2
	}
	return 0
}
