// Package autosave keeps <data>/autosave.scadabench current without making a mutation wait for
// the full archive: every committed mutation appends one synced line to the change journal
// (Journal), and a coalescing background writer (SnapshotWriter) writes full snapshots.
// Fold applies a journal left by a process that died, so Python never sees a journal.
package autosave

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"studio-engine/internal/store/projectfile"
)

// File names inside the data folder (persistence-2 section 1; the journal is Go-only).
const (
	AutosaveName = projectfile.AutosaveName
	TempName     = "autosave.tmp" // Path("autosave.scadabench").with_suffix(".tmp")
	JournalName  = "autosave.journal"
)

// syncFile and renameFile are (*os.File).Sync and os.Rename; tests replace them to check
// that a file's data is flushed before it is renamed into place.
var (
	syncFile   = (*os.File).Sync
	renameFile = os.Rename
)

// WriteAutosave writes payload like ProjectService.save(): <data>/autosave.tmp
// (create/truncate, then fsync so a power cut cannot leave an empty file behind the
// rename) then os.replace onto autosave.scadabench (MoveFileEx with
// MOVEFILE_REPLACE_EXISTING, which os.Rename uses on Windows). The bytes are unchanged.
func WriteAutosave(dataDir string, payload []byte) error {
	temporary := filepath.Join(dataDir, TempName)
	if err := writeSynced(temporary, payload); err != nil {
		return err
	}
	return renameFile(temporary, filepath.Join(dataDir, AutosaveName))
}

// writeSynced creates or truncates path, writes data and fsyncs it before closing.
func writeSynced(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o666)
	if err != nil {
		return err
	}
	_, err = file.Write(data)
	if err == nil {
		err = syncFile(file)
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

// autosaveDigest returns the autosave bytes and their SHA-256 hex; a missing file gives
// (nil, "", nil).
func autosaveDigest(dataDir string) ([]byte, string, error) {
	payload, err := os.ReadFile(filepath.Join(dataDir, AutosaveName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", err
	}
	return payload, digest(payload), nil
}

func digest(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
