// Package nativetest finds the native programs for wire tests, like the Python protocol
// tests: STUDIO_NATIVE_BIN, else studio/native/build/Release of this checkout. Tests skip
// when the program is not built.
package nativetest

import (
	"os"
	"path/filepath"

	"testing"
)

// Dir is the folder that holds the native programs.
func Dir() string {
	if folder := os.Getenv("STUDIO_NATIVE_BIN"); folder != "" {
		return folder
	}
	return filepath.Join(StudioDir(), "native", "build", "Release")
}

// StudioDir is the studio/ folder: the parent of the Go module root (studio/engine), found
// from the test's working folder so -trimpath builds work too.
func StudioDir() string {
	folder, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(folder, "go.mod")); err == nil {
			return filepath.Dir(folder)
		}
		parent := filepath.Dir(folder)
		if parent == folder {
			return folder
		}
		folder = parent
	}
}

// Program returns the path of name, or skips the test when it is not built.
func Program(t testing.TB, name string) string {
	t.Helper()
	path := filepath.Join(Dir(), name)
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s is not built (%s); set STUDIO_NATIVE_BIN", name, path)
	}
	return path
}
