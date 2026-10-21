package recovery

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"syscall"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/store/pytext"
)

// Restore reopens an autosave like ProjectService.restore(payload, persist=False); its
// error text becomes the "details" of an unreadable result (Python's str(error)).
type Restore func(payload []byte) error

// Startup results and texts of startup_recovery.recover_startup.
const (
	restoredMessage   = "Recovered the last saved substation. All IEDs remain stopped."
	unreadableMessage = "The last autosave could not be reopened. " +
		"A separate recovery copy was preserved before allowing new work."
)

// RecoverStartup is recover_startup: it returns nil when there is no autosave,
// {"status": "restored", "message"} after a successful restore, or {"status":
// "unreadable", "message", "details", "backup"} after copying the unreadable autosave to
// recovery/unreadable-autosave-<uuid>.subsim. Errors stop startup (RuntimeError texts:
// "Autosave is not a regular local file; inspect it before starting ScadaBench",
// "Recovery folder is a link; cannot safely preserve unreadable autosave", "Cannot
// preserve unreadable autosave; startup stopped to protect it"). Fold the autosave journal
// before calling it.
func RecoverStartup(dataDir string, restore Restore) (*pyjson.Object, error) {
	source := pytext.Path(dataDir, "autosave.subsim")
	if _, err := os.Stat(source); err != nil {
		if missing(err) {
			return nil, nil
		}
		return nil, err
	}
	if info, err := os.Lstat(source); err != nil || !info.Mode().IsRegular() {
		return nil, runtimeError("Autosave is not a regular local file; inspect it before starting ScadaBench")
	}
	details := reopen(source, restore)
	if details == nil {
		return pyjson.NewObject().Set("status", "restored").Set("message", restoredMessage), nil
	}
	folder, err := prepare(dataDir, "Recovery folder is a link; cannot safely preserve unreadable autosave", runtimeError)
	if err != nil {
		return nil, err
	}
	backup := pytext.Path(folder, uniqueName("unreadable-autosave-"))
	if err := copyUnreadable(source, backup); err != nil {
		return nil, runtimeError("Cannot preserve unreadable autosave; startup stopped to protect it")
	}
	return pyjson.NewObject().Set("status", "unreadable").Set("message", unreadableMessage).
		Set("details", *details).Set("backup", backup), nil
}

// reopen returns nil on success or Python's str() of the failure.
func reopen(source string, restore Restore) *string {
	text := func(message string) *string { return &message }
	info, err := os.Stat(source)
	if err != nil {
		return text(pytext.OSError(err, false))
	}
	if info.Size() > archive.MaxUnpacked {
		return text("Saved archive exceeds the 100 MB project limit")
	}
	payload, err := os.ReadFile(source)
	if err != nil {
		return text(pytext.OSError(err, true))
	}
	if err := restore(payload); err != nil {
		return text(err.Error())
	}
	return nil
}

// missing mirrors the errors Path.exists() treats as "does not exist": not found, not a
// directory, and the Windows codes 21 (not ready), 123 (invalid name), 1921 (link loop).
func missing(err error) bool {
	var code syscall.Errno
	if errors.As(err, &code) && (code == 21 || code == 123 || code == 1921 || code == 267) {
		return true
	}
	return errors.Is(err, fs.ErrNotExist)
}

// copyUnreadable copies the unreadable autosave. It is copyFile; only the parity test build
// (copy_seam_parity.go) can replace it, for the shutil.copyfile fault of tests/parity.
var copyUnreadable = copyFile

// copyFile is shutil.copyfile: create or truncate the destination and copy the bytes.
func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}
	return err
}
