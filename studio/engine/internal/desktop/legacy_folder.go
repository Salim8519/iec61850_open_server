package desktop

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// Folder names of legacy_folder.py: the pre-ScadaBench and current user folders.
const (
	LegacyFolderName = "SubstationStudio"
	FolderName       = "ScadaBench"
)

// MigrateLegacyFolder is legacy_folder.migrate (packaged launches only): move
// %LOCALAPPDATA%\SubstationStudio to %LOCALAPPDATA%\ScadaBench when only the old folder
// exists. It returns "moved", "kept-both" (both exist; nothing merged or deleted) or
// "nothing" (no old folder, empty localAppData, or the move failed).
//
// Python's Path.exists raised on errors other than "not found"; this port treats such a
// new folder as present ("kept-both") so it never moves onto something it cannot see.
func MigrateLegacyFolder(localAppData string) string {
	if localAppData == "" {
		return "nothing"
	}
	old := filepath.Join(localAppData, LegacyFolderName)
	current := filepath.Join(localAppData, FolderName)
	if info, err := os.Stat(old); err != nil || !info.IsDir() {
		return "nothing"
	}
	if _, err := os.Stat(current); err == nil || !errors.Is(err, fs.ErrNotExist) {
		return "kept-both"
	}
	if err := os.Rename(old, current); err != nil {
		return "nothing"
	}
	return "moved"
}
