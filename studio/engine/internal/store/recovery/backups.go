package recovery

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"io"
	"os"

	"studio-engine/internal/apperr"
	"studio-engine/internal/store/pytext"
)

// PreserveMigration is migration_backup.preserve: the exact bytes of a legacy archive go to
// recovery/before-format-8-migration-<uuid>.subsim (overwrite allowed), are read back and
// compared, and the archive is tested like ZipFile.testzip(). Errors (StudioError):
// "Recovery directory must not be a symbolic link", "Project migration recovery
// verification failed", "Project migration recovery archive is corrupt", "Project
// migration recovery archive is invalid"; file system failures are returned as they are.
func PreserveMigration(dataDir string, payload []byte) (string, error) {
	folder, err := prepare(dataDir, "Recovery directory must not be a symbolic link", studioError)
	if err != nil {
		return "", err
	}
	destination := pytext.Path(folder, uniqueName("before-format-8-migration-"))
	if err := os.WriteFile(destination, payload, 0o666); err != nil {
		return "", err
	}
	stored, err := os.ReadFile(destination)
	if err != nil {
		return "", err
	}
	if !bytes.Equal(stored, payload) {
		return "", apperr.New("Project migration recovery verification failed")
	}
	reader, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		return "", apperr.New("Project migration recovery archive is invalid")
	}
	if !testZip(reader) {
		return "", apperr.New("Project migration recovery archive is corrupt")
	}
	return destination, nil
}

// testZip reads every member to the end so archive/zip checks sizes and CRC-32 values.
// Python's testzip also reports a damaged member; corrupt deflate data raises zlib.error
// there, while it counts as a damaged member here.
func testZip(reader *zip.Reader) bool {
	for _, member := range reader.File {
		stream, err := member.Open()
		if err != nil {
			return false
		}
		_, err = io.Copy(io.Discard, stream)
		_ = stream.Close()
		if err != nil {
			return false
		}
	}
	return true
}

// WriteDeleteBackup is the backup step of bulk_removal.remove: payload (service.export(),
// the current project archive) is written to recovery/before-ied-delete-<uuid>.subsim with
// exclusive create, then verified by SHA-256. Errors (StudioError): "Recovery directory
// must not be a symbolic link", "Recovery backup verification failed; nothing deleted".
func WriteDeleteBackup(dataDir string, payload []byte) (string, error) {
	folder, err := prepare(dataDir, "Recovery directory must not be a symbolic link", studioError)
	if err != nil {
		return "", err
	}
	backup := pytext.Path(folder, uniqueName("before-ied-delete-"))
	file, err := os.OpenFile(backup, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o666)
	if err != nil {
		return "", err
	}
	_, err = file.Write(payload)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", err
	}
	stored, err := os.ReadFile(backup)
	if err != nil {
		return "", err
	}
	if sha256.Sum256(stored) != sha256.Sum256(payload) {
		return "", apperr.New("Recovery backup verification failed; nothing deleted")
	}
	return backup, nil
}

func studioError(message string) error { return apperr.New(message) }

func runtimeError(message string) error { return apperr.RuntimeError(message) }
