package netcfg

import (
	"strconv"
	"strings"
	"time"

	"studio-engine/internal/pyjson"
)

// batchSize keeps one PowerShell script well under the Windows command-line limit.
const batchSize = 20

// batchMarker starts the line the batch script writes after each change.
const batchMarker = "#scadabench-change "

// ExecuteBatch runs the request's changes like ExecuteRequest, with the same commands in the
// same order, but up to 20 of them in one PowerShell start (ADR-0131). One start loads the
// networking modules once: 9 changes took 2 minutes one by one on the owner's PC. A change whose
// line is missing (PowerShell crashed or timed out) runs again on its own, as before.
func ExecuteBatch(request *pyjson.Object, executor Executor) ([]any, error) {
	items, _ := get(request, "items").([]any)
	commands, keys := make([]string, len(items)), make([]string, len(items))
	for index, raw := range items {
		item, _ := raw.(*pyjson.Object)
		var err error
		if get(request, "action") == "apply" {
			commands[index], err = ApplyCommand(item)
		} else {
			commands[index], err = RemoveCommand(item)
		}
		if err != nil {
			return nil, err
		}
		if keys[index], err = ItemKey(item); err != nil {
			return nil, err
		}
	}
	results := []any{}
	for start := 0; start < len(commands); start += batchSize {
		end := min(start+batchSize, len(commands))
		done := runBatch(executor, commands[start:end])
		for index := start; index < end; index++ {
			outcome, found := done[index-start]
			if !found {
				ok, message, err := runChange(executor, commands[index])
				if err != nil {
					return nil, err
				}
				outcome = batchOutcome{ok, message}
			}
			results = append(results, pyjson.NewObject().Set("key", keys[index]).Set("ok", outcome.ok).
				Set("message", cutRunes(outcome.message, 500)))
		}
	}
	return results, nil
}

type batchOutcome struct {
	ok      bool
	message string
}

// BatchScript is the PowerShell script for a batch: each change in its own try/catch, so one
// failure does not stop the others, followed by a marker line with its number and outcome.
func BatchScript(commands []string) string {
	var script strings.Builder
	script.WriteString("$ErrorActionPreference='Stop'; ")
	for index, command := range commands {
		number := strconv.Itoa(index)
		script.WriteString("try { " + command + " | Out-Null; '" + batchMarker + number + " ok' } catch { '" +
			batchMarker + number + " fail ' + ($_.Exception.Message -replace '\\s+', ' ') }; ")
	}
	return script.String()
}

// runBatch starts one PowerShell for the commands and reads back each change's outcome. A
// change without a marker line is left out; the caller runs it again on its own.
func runBatch(executor Executor, commands []string) map[int]batchOutcome {
	done := map[int]batchOutcome{}
	timeout := HelperCommandTimeout + time.Duration(len(commands))*15*time.Second
	output, err := executor.Run(Command{Argv: PowerShell(BatchScript(commands)), Timeout: timeout})
	if err != nil {
		return done
	}
	for _, line := range strings.Split(output.Stdout, "\n") {
		rest, found := strings.CutPrefix(strings.TrimSpace(line), batchMarker)
		if !found {
			continue
		}
		fields := strings.SplitN(rest, " ", 3)
		number, err := strconv.Atoi(fields[0])
		if err != nil || number < 0 || number >= len(commands) || len(fields) < 2 {
			continue
		}
		message := ""
		if len(fields) == 3 {
			message = strings.TrimSpace(fields[2])
		}
		done[number] = batchOutcome{fields[1] == "ok", message}
	}
	return done
}
