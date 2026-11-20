package service

import (
	"strings"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/worker"
)

// failingOperator answers every operate with err, as studio_control does when it is frozen
// (a *worker.Timeout after 30 s) or cannot start.
type failingOperator struct{ err error }

func (o failingOperator) Operate(string, any, string, any, any) (*pyjson.Object, error) {
	return nil, o.err
}

// lastLogAfterCommand runs one SCADA command and returns its answer and its command-log line.
func lastLogAfterCommand(t *testing.T, err error) (answer, line *pyjson.Object) {
	t.Helper()
	process := &fakeGatewayProcess{}
	manager := runningGateway(t, &slowOperator{}, process)
	manager.control = failingOperator{err}
	manager.event(scadaCommand("t1"))
	deadline := time.Now().Add(2 * time.Second)
	for len(process.results()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	results := process.results()
	if len(results) != 1 {
		t.Fatalf("answers %v", results)
	}
	entries := get(manager.Commands.After(0, 200), "entries").([]any)
	return results[0], entries[len(entries)-1].(*pyjson.Object)
}

// A command whose control program never answered may still have operated the device: the log
// line and the SCADA answer say "result unknown", never an empty or plain failed line (fix plan
// A8 in the gateway; found live on Windows: the log line stayed empty).
func TestGatewayCommandTimeoutIsResultUnknown(t *testing.T) {
	answer, line := lastLogAfterCommand(t, &worker.Timeout{Op: "operate"})
	want := "Bay did not answer in time; result unknown — check the position before trying again"
	if get(answer, "ok") != false || get(answer, "reason") != want {
		t.Fatalf("SCADA answer %v, want ok false and %q", answer, want)
	}
	if get(line, "applied") != "result unknown: "+want {
		t.Fatalf("command log line %v", line)
	}
}

// Any other control failure is written to the log line too, so the page never shows a command
// without an outcome.
func TestGatewayCommandFailureIsLogged(t *testing.T) {
	text := "The control program (studio_control.exe) is not built yet; rebuild the native programs"
	answer, line := lastLogAfterCommand(t, apperr.RuntimeError(text))
	if get(answer, "reason") != text || !strings.HasPrefix(pyText(get(line, "applied")), "failed: "+text) {
		t.Fatalf("answer %v, log line %v", answer, line)
	}
}

func pyText(value any) string {
	text, _ := value.(string)
	return text
}
