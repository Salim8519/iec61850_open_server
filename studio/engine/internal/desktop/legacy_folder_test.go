package desktop

import (
	"os"
	"path/filepath"
	"testing"

	"studio-engine/internal/testutil"
)

func TestMigrateLegacyFolder(t *testing.T) {
	if got := MigrateLegacyFolder(""); got != "nothing" {
		t.Fatalf("no LOCALAPPDATA: %s", got)
	}
	local := testutil.TempDir(t)
	if got := MigrateLegacyFolder(local); got != "nothing" {
		t.Fatalf("no old folder: %s", got)
	}
	old := filepath.Join(local, LegacyFolderName)
	current := filepath.Join(local, FolderName)
	if err := os.WriteFile(old, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := MigrateLegacyFolder(local); got != "nothing" {
		t.Fatalf("old name is a file: %s", got)
	}
	if err := os.Remove(old); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(old, "projects"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(old, "license.json"), []byte("lic"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := MigrateLegacyFolder(local); got != "moved" {
		t.Fatalf("only old folder: %s", got)
	}
	if data, err := os.ReadFile(filepath.Join(current, "license.json")); err != nil || string(data) != "lic" {
		t.Fatalf("moved licence: %q %v", data, err)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatal("old folder must be gone after the move")
	}
	if err := os.MkdirAll(old, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := MigrateLegacyFolder(local); got != "kept-both" {
		t.Fatalf("both folders: %s", got)
	}
	if _, err := os.Stat(old); err != nil {
		t.Fatal("kept-both must leave the old folder")
	}
}
