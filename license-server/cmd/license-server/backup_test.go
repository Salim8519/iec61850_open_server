package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"substation-license-server/internal/store"
)

func TestBackupThenRestoreGivesTheSameFiles(t *testing.T) {
	source, target := t.TempDir(), filepath.Join(t.TempDir(), "restored")
	db, err := store.Open(dbPath(source))
	if err != nil {
		t.Fatal(err)
	}
	db.CreateLicense(store.License{Key: "SUBS-BACK-UPTE-STKE-YAAA", Type: "lifetime"})
	db.Close()
	for _, name := range backupFiles {
		if err := os.WriteFile(filepath.Join(source, name), []byte("content of "+name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(t.TempDir(), "backup.tar.gz")
	if err := backup([]string{"--data", source, "--out", out}); err != nil {
		t.Fatal(err)
	}
	if err := restore([]string{"--data", target, "--from", out}); err != nil {
		t.Fatal(err)
	}
	for _, name := range backupFiles {
		before, _ := os.ReadFile(filepath.Join(source, name))
		after, err := os.ReadFile(filepath.Join(target, name))
		if err != nil || !bytes.Equal(before, after) {
			t.Fatalf("%s not restored the same: %v", name, err)
		}
	}
	restored, err := store.Open(dbPath(target))
	if err != nil {
		t.Fatal(err)
	}
	defer restored.Close()
	if _, err := restored.GetLicense("SUBS-BACK-UPTE-STKE-YAAA"); err != nil {
		t.Fatalf("restored database lost the key: %v", err)
	}
	if err := restore([]string{"--data", target, "--from", out}); err == nil {
		t.Fatal("restore must never overwrite an existing data folder")
	}
}
