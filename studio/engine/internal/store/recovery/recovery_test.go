package recovery

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
)

func message(err error) string {
	if appErr, ok := err.(*apperr.Error); ok {
		return appErr.Message
	}
	if err == nil {
		return ""
	}
	return "other: " + err.Error()
}

func validArchive(t *testing.T) []byte {
	payload, err := archive.Encode(pyjson.NewObject().Set("format", int64(9)))
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestPreserveMigrationKeepsExactBytes(t *testing.T) {
	dir := t.TempDir()
	payload := validArchive(t)
	path, err := PreserveMigration(dir, payload)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Base(path)
	if filepath.Dir(path) != filepath.Join(dir, "recovery") || !strings.HasPrefix(name, "before-format-8-migration-") ||
		len(name) != len("before-format-8-migration-")+32+len(".subsim") || strings.Contains(path, "/") {
		t.Fatalf("path %s", path)
	}
	if stored, _ := os.ReadFile(path); !bytes.Equal(stored, payload) {
		t.Fatal("bytes differ")
	}
	if _, err := PreserveMigration(dir, []byte("not a zip")); message(err) != "Project migration recovery archive is invalid" {
		t.Errorf("invalid: %v", err)
	}
	broken := bytes.Clone(payload)
	central := bytes.LastIndex(broken, []byte{0x50, 0x4b, 0x01, 0x02})
	broken[central+16] ^= 0xff // CRC in the central directory
	if _, err := PreserveMigration(dir, broken); message(err) != "Project migration recovery archive is corrupt" {
		t.Errorf("corrupt: %v", err)
	}
}

func TestDeleteBackupIsVerified(t *testing.T) {
	dir := t.TempDir()
	payload := validArchive(t)
	path, err := WriteDeleteBackup(dir, payload)
	if err != nil || !strings.HasPrefix(filepath.Base(path), "before-ied-delete-") {
		t.Fatalf("%s %v", path, err)
	}
	if stored, _ := os.ReadFile(path); !bytes.Equal(stored, payload) {
		t.Fatal("bytes differ")
	}
}

func TestRecoveryFolderLinkIsRefused(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir()
	if err := os.Symlink(target, filepath.Join(dir, "recovery")); err != nil {
		t.Skipf("symbolic links need extra rights here: %v", err)
	}
	if _, err := WriteDeleteBackup(dir, []byte("x")); message(err) != "Recovery directory must not be a symbolic link" {
		t.Errorf("delete backup: %v", err)
	}
	_ = os.WriteFile(filepath.Join(dir, "autosave.subsim"), []byte("bad"), 0o666)
	_, err := RecoverStartup(dir, func([]byte) error { return errors.New("bad") })
	if appErr, ok := err.(*apperr.Error); !ok || appErr.Kind != apperr.Runtime ||
		appErr.Message != "Recovery folder is a link; cannot safely preserve unreadable autosave" {
		t.Errorf("startup: %v", err)
	}
}

func TestRecoverStartupResults(t *testing.T) {
	dir := t.TempDir()
	called := false
	result, err := RecoverStartup(dir, func([]byte) error { called = true; return nil })
	if result != nil || err != nil || called {
		t.Fatalf("no autosave: %v %v", result, err)
	}
	payload := validArchive(t)
	_ = os.WriteFile(filepath.Join(dir, "autosave.subsim"), payload, 0o666)
	result, err = RecoverStartup(dir, func(got []byte) error {
		if !bytes.Equal(got, payload) {
			t.Error("restore got other bytes")
		}
		return nil
	})
	if encoded, _ := pyjson.Dumps(result); err != nil || string(encoded) !=
		`{"status": "restored", "message": "Recovered the last saved substation. All IEDs remain stopped."}` {
		t.Fatalf("restored: %s %v", encoded, err)
	}
	result, err = RecoverStartup(dir, func([]byte) error { return apperr.New("Unsupported project format") })
	if err != nil {
		t.Fatal(err)
	}
	backup, _ := result.Get("backup")
	details, _ := result.Get("details")
	status, _ := result.Get("status")
	if status != "unreadable" || details != "Unsupported project format" || result.Keys()[3] != "backup" {
		t.Fatalf("unreadable: %v", result.Keys())
	}
	if copied, _ := os.ReadFile(backup.(string)); !bytes.Equal(copied, payload) ||
		!strings.HasPrefix(filepath.Base(backup.(string)), "unreadable-autosave-") {
		t.Fatal("backup copy differs")
	}
}

func TestRecoverStartupSizeLimitAndNonFile(t *testing.T) {
	dir := t.TempDir()
	file, _ := os.Create(filepath.Join(dir, "autosave.subsim"))
	_ = file.Truncate(archive.MaxUnpacked + 1) // sparse: no real 100 MB written
	_ = file.Close()
	result, err := RecoverStartup(dir, func([]byte) error { t.Error("restore called"); return nil })
	if details, _ := result.Get("details"); err != nil || details != "Saved archive exceeds the 100 MB project limit" {
		t.Fatalf("size: %v %v", details, err)
	}
	other := t.TempDir()
	_ = os.Mkdir(filepath.Join(other, "autosave.subsim"), 0o777)
	_, err = RecoverStartup(other, nil)
	if message(err) != "Autosave is not a regular local file; inspect it before starting ScadaBench" {
		t.Fatalf("directory autosave: %v", err)
	}
}

func TestBackgroundRecovery(t *testing.T) {
	release := make(chan struct{})
	var logged error
	background := StartBackground(func() (*pyjson.Object, error) {
		<-release
		return nil, errors.New("stopped")
	}, func(err error) { logged = err })
	if background.Wait(10*time.Millisecond) || func() bool { _, err := background.Result(); return err != nil }() {
		t.Fatal("finished too early")
	}
	close(release)
	if !background.Wait(-1) {
		t.Fatal("wait forever returned false")
	}
	if _, err := background.Result(); err == nil || logged == nil {
		t.Fatal("error lost")
	}
}
