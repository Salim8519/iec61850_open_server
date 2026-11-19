package worker

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// A worker that stops reading cannot hang Request or Close: the write has the request's
// deadline and the worker is killed when it is missed (fix plan A10). The frozen worker is a
// shell script, so the test runs outside Windows only.
func TestCloseReturnsWhenWorkerStopsReading(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the frozen worker is a shell script")
	}
	path := filepath.Join(t.TempDir(), "deaf_worker")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	process, err := Spawn(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	big := pyjson.NewObject().Set("blob", strings.Repeat("x", 1<<20)) // more than a pipe holds
	started := time.Now()
	_, err = process.Request("load", 300*time.Millisecond, big)
	var timeout *Timeout
	if !errors.As(err, &timeout) || time.Since(started) > 2*time.Second {
		t.Fatalf("stuck write: %v after %s", err, time.Since(started))
	}
	closed := make(chan struct{})
	go func() { process.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("Close hangs on a worker that does not read")
	}
	if process.Alive() {
		t.Fatal("the frozen worker is still running")
	}
}
