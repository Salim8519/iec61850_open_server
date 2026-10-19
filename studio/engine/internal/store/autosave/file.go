// Package autosave keeps <data>/autosave.subsim current without making a mutation wait for
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
)

// File names inside the data folder (persistence-2 section 1; the journal is Go-only).
const (
	AutosaveName = "autosave.subsim"
	TempName     = "autosave.tmp" // Path("autosave.subsim").with_suffix(".tmp")
	JournalName  = "autosave.journal"
)

// WriteAutosave writes payload like ProjectService.save(): <data>/autosave.tmp
// (create/truncate, no fsync) then os.replace onto autosave.subsim (MoveFileEx with
// MOVEFILE_REPLACE_EXISTING, which os.Rename uses on Windows).
func WriteAutosave(dataDir string, payload []byte) error {
	temporary := filepath.Join(dataDir, TempName)
	if err := os.WriteFile(temporary, payload, 0o666); err != nil {
		return err
	}
	return os.Rename(temporary, filepath.Join(dataDir, AutosaveName))
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
