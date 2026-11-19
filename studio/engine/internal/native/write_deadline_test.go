package native

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// deafWorker is a program that never reads its stdin (a frozen worker). It needs a POSIX
// shell, so the test runs outside Windows only.
func deafWorker(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the frozen worker is a shell script")
	}
	path := filepath.Join(t.TempDir(), "deaf_worker")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexec sleep 60\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// A worker that stops reading cannot hang a request or Close: the write has the request's
// deadline and the worker is killed when it is missed (fix plan A10).
func TestRequestAndCloseReturnWhenWorkerStopsReading(t *testing.T) {
	w, err := Start(deafWorker(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	big := pyjson.NewObject().Set("blob", strings.Repeat("x", 1<<20)) // more than a pipe holds
	go func() { _, _ = w.Request("load", time.Hour, big) }()            // holds the pipe forever before the fix
	time.Sleep(100 * time.Millisecond)
	started := time.Now()
	_, err = w.Request("hello", 300*time.Millisecond, nil)
	if !IsTimeout(err) || time.Since(started) > 2*time.Second {
		t.Fatalf("request behind a stuck write: %v after %s", err, time.Since(started))
	}
	closed := make(chan struct{})
	go func() { w.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatal("Close hangs on a worker that does not read")
	}
	if w.Alive() {
		t.Fatal("the frozen worker is still running")
	}
}
