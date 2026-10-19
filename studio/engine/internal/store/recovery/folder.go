// Package recovery writes the <data>/recovery/*.subsim copies (persistence-3 section 1) and
// runs startup recovery exactly as studio_app.projects.startup_recovery and
// background_recovery do. Returned paths are str(Path) strings (backslashes, as given).
package recovery

import (
	"crypto/rand"
	"encoding/hex"
	"io/fs"
	"os"

	"studio-engine/internal/store/pytext"
)

// FolderName is the recovery folder inside the data folder.
const FolderName = "recovery"

// isSymlink is Path.is_symlink(): a symbolic link (a junction is not one in Python 3.12);
// any lstat failure counts as "not a link".
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&fs.ModeSymlink != 0
}

// makeFolder is Path.mkdir(exist_ok=True) without parents: an existing directory is fine,
// anything else re-raises the original error.
func makeFolder(path string) error {
	err := os.Mkdir(path, 0o777)
	if err == nil {
		return nil
	}
	if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
		return nil
	}
	return err
}

// uniqueName returns prefix + uuid4().hex + ".subsim".
func uniqueName(prefix string) string {
	var random [16]byte
	_, _ = rand.Read(random[:])
	random[6] = random[6]&0x0f | 0x40 // version 4
	random[8] = random[8]&0x3f | 0x80 // RFC 4122 variant
	return prefix + hex.EncodeToString(random[:]) + ".subsim"
}

// prepare returns the recovery folder path after the link check and mkdir.
func prepare(dataDir, linkMessage string, linkError func(string) error) (string, error) {
	folder := pytext.Path(dataDir, FolderName)
	if isSymlink(folder) {
		return "", linkError(linkMessage)
	}
	if err := makeFolder(folder); err != nil {
		return "", err
	}
	return folder, nil
}
