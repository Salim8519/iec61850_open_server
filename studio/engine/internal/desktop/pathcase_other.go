//go:build !windows

package desktop

import "path/filepath"

// abspath is posixpath.abspath: the cleaned absolute path.
func abspath(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return absolute
}

// normcase is posixpath.normcase, which leaves the path unchanged.
func normcase(path string) string { return path }
