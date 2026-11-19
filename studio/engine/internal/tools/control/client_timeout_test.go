package control

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/worker"
)

// silentControl is a studio_control whose operate never answers in time.
type silentControl struct {
	killed, closed bool
}

func (p *silentControl) Request(op string, _ time.Duration, _ *pyjson.Object) (any, error) {
	if op == "hello" {
		return pyjson.NewObject().Set("capabilities", []any{"control_operate", "typed_commands"}), nil
	}
	return nil, &worker.Timeout{Op: op}
}
func (p *silentControl) Alive() bool { return !p.killed && !p.closed }
func (p *silentControl) Close()      { p.closed = true }
func (p *silentControl) Kill()       { p.killed = true }

// A command with no answer in time kills the control worker, so it can never complete the
// command later, and the next command starts a fresh worker (fix plan A8).
func TestOperateTimeoutKillsTheWorker(t *testing.T) {
	executable := filepath.Join(t.TempDir(), "studio_control.exe")
	if err := os.WriteFile(executable, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var spawned []*silentControl
	client := NewClient(executable, func(string, func(*pyjson.Object)) (worker.Process, error) {
		process := &silentControl{}
		spawned = append(spawned, process)
		return process, nil
	})
	_, err := client.Operate("127.0.0.1", int64(102), "LD/CSWI1.Pos", true, nil)
	var timeout *worker.Timeout
	if !errors.As(err, &timeout) {
		t.Fatalf("error %v, want the timeout", err)
	}
	if len(spawned) != 1 || !spawned[0].killed || !spawned[0].closed {
		t.Fatalf("the silent worker is still there: %+v", spawned)
	}
	_, _ = client.Typed("typed_operate", pyjson.NewObject())
	if len(spawned) != 2 || !spawned[1].killed {
		t.Fatalf("the next command did not start a fresh worker, or kept a silent one: %+v", spawned)
	}
}
