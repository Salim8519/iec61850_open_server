//go:build windows

package desktop

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestStartBrowserReapsProcess starts a harmless console command (never a browser) to
// check the Popen equivalent: new process group, PID known, Exited after it ends.
func TestStartBrowserReapsProcess(t *testing.T) {
	shell := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	started, err := startBrowser([]string{shell, "/c", "exit", "0"})
	if err != nil {
		t.Fatal(err)
	}
	if started.PID() <= 0 {
		t.Fatalf("pid %d", started.PID())
	}
	finished := make(chan struct{})
	go func() { started.Wait(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(10 * time.Second):
		t.Fatal("process never reaped")
	}
	if !started.Exited() {
		t.Fatal("Exited must be true after Wait")
	}
	if _, err := startBrowser([]string{filepath.Join(t.TempDir(), "missing.exe")}); err == nil {
		t.Fatal("a missing browser must fail to start (AppWindow then uses the default browser)")
	}
}
