// Package recovery writes the <data>/recovery/*.scadabench copies (persistence-3 section 1) and
// runs startup recovery exactly as studio_app.projects.startup_recovery and
// background_recovery do. Returned paths are str(Path) strings (backslashes, as given).
package recovery

import (
	"io/fs"
	"os"

	"studio-engine/internal/store/projectfile"
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

// uniqueName returns prefix + uuid4().hex + ".scadabench".
func uniqueName(prefix string) string { return projectfile.UniqueName(prefix) }

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
