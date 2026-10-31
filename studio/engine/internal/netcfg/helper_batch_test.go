package netcfg

import (
	"runtime"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func aliasRequest(ips ...string) *pyjson.Object {
	items := []any{}
	for _, ip := range ips {
		items = append(items, pyjson.NewObject().Set("kind", "alias").Set("adapter_guid", "{E}").
			Set("adapter_index", 15).Set("adapter_name", "Ethernet").Set("ip", ip).Set("prefix", 24).
			Set("owned", true).Set("action", "add"))
	}
	return pyjson.NewObject().Set("action", "apply").Set("items", items)
}

func TestBatchRunsAllChangesInOnePowerShell(t *testing.T) {
	calls := []Command{}
	fake := ExecutorFunc(func(command Command) (Output, error) {
		calls = append(calls, command)
		if len(calls) == 1 {
			return Output{0, batchMarker + "0 ok\r\n" + batchMarker + "1 fail The object already exists.\r\n", ""}, nil
		}
		return Output{0, "", ""}, nil
	})
	results, err := ExecuteBatch(aliasRequest("10.0.0.1", "10.0.0.2", "10.0.0.3"), fake)
	if err != nil {
		t.Fatal(err)
	}
	script := calls[0].Argv[len(calls[0].Argv)-1]
	if strings.Count(script, "New-NetIPAddress") != 3 || !strings.Contains(script, "-IPAddress 10.0.0.3 ") {
		t.Fatalf("the batch does not hold every change: %s", script)
	}
	// The third change wrote no line, so it ran again on its own, exactly as before.
	if len(calls) != 2 || calls[1].Argv[len(calls[1].Argv)-1] != ChangeArgv(mustApply(t, "10.0.0.3"))[4] {
		t.Fatalf("missing change not retried alone: %d calls", len(calls))
	}
	text := dumpsText(t, results)
	for _, want := range []string{`"ok": true`, `"message": "The object already exists."`, "10.0.0.3"} {
		if !strings.Contains(text, want) {
			t.Fatalf("results %s lack %s", text, want)
		}
	}
}

func mustApply(t *testing.T, ip string) string {
	items, _ := get(aliasRequest(ip), "items").([]any)
	command, err := ApplyCommand(items[0].(*pyjson.Object))
	if err != nil {
		t.Fatal(err)
	}
	return command
}

func TestBatchScriptInRealPowerShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell runs on Windows")
	}
	commands := []string{"Get-Item $env:SystemRoot", "& { throw 'first  problem' }", "Get-Date", "Write-Error 'second'"}
	done := runBatch(SystemExecutor{}, commands)
	want := map[int]batchOutcome{0: {true, ""}, 1: {false, "first problem"}, 2: {true, ""}, 3: {false, "second"}}
	for number, outcome := range want {
		if done[number] != outcome {
			t.Errorf("change %d: %+v want %+v", number, done[number], outcome)
		}
	}
}
