//go:build parity

package recovery

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/testutil"
)

// The parity copy seam stands in for shutil.copyfile: a failing copy stops startup with
// Python's text and leaves the damaged autosave as it was; "original" copies for real.
func TestCopySeamFailureProtectsAutosave(t *testing.T) {
	defer func() { copyUnreadable = copyFile }()
	dir := testutil.TempDir(t)
	source := filepath.Join(dir, "autosave.scadabench")
	if err := os.WriteFile(source, []byte("damaged"), 0o666); err != nil {
		t.Fatal(err)
	}
	var seen []string
	SetCopySeam(func(from, to string, _ func(string, string) error) error {
		seen = append(seen, from, to)
		return errors.New("injected backup failure")
	})
	_, err := RecoverStartup(dir, func([]byte) error { return errors.New("bad") })
	if message(err) != "Cannot preserve unreadable autosave; startup stopped to protect it" {
		t.Fatalf("error: %v", err)
	}
	if len(seen) != 2 || seen[0] != source || filepath.Dir(seen[1]) != filepath.Join(dir, FolderName) {
		t.Fatalf("seam arguments: %v", seen)
	}
	if data, _ := os.ReadFile(source); string(data) != "damaged" {
		t.Fatal("the autosave must stay as it was")
	}
	SetCopySeam(func(from, to string, original func(string, string) error) error { return original(from, to) })
	result, err := RecoverStartup(dir, func([]byte) error { return errors.New("bad") })
	if status, _ := result.Get("status"); err != nil || status != "unreadable" {
		t.Fatalf("original copy: %v %v", result, err)
	}
}
