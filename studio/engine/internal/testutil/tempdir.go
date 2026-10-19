// Package testutil holds helpers shared by tests only; production code never imports it.
package testutil

import (
	"os"
	"testing"
	"time"
)

// TempDir is t.TempDir with a patient cleanup: on Windows a just-closed or just-renamed
// file can stay "delete pending" for a moment while a scanner holds it, and the folder
// removal then fails with "The directory is not empty" although nothing is left.
func TempDir(t testing.TB) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "studio-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		var err error
		for attempt := 0; attempt < 50; attempt++ {
			if err = os.RemoveAll(dir); err == nil {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Errorf("remove %s: %v", dir, err)
	})
	return dir
}
