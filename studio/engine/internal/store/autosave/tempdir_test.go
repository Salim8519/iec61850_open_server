package autosave

import (
	"os"
	"testing"
	"time"
)

// tempDir is t.TempDir with a patient cleanup: on Windows a just-closed or just-renamed
// file can stay "delete pending" for a moment while a scanner holds it, and the folder
// removal then fails with "The directory is not empty" although nothing is left.
func tempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "autosave-test-")
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
