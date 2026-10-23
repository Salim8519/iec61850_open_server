package xsd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"studio-engine/internal/testutil"
)

// Change 0185 item 11: the work file reaches the helper as an extended-length path.
func TestExtendedPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		if extendedPath("/tmp/a") != "/tmp/a" {
			t.Fatal("non-Windows paths must stay unchanged")
		}
		return
	}
	for path, want := range map[string]string{
		`C:\data\cache\xsd\work\source-1.scl`: `\\?\C:\data\cache\xsd\work\source-1.scl`,
		`\\server\share\work\source-1.scl`:    `\\?\UNC\server\share\work\source-1.scl`,
		`\\?\C:\already`:                      `\\?\C:\already`,
	} {
		if got := extendedPath(path); got != want {
			t.Errorf("extendedPath(%q) = %q, want %q", path, got, want)
		}
	}
}

// A data folder deeper than MAX_PATH validates with the real helper (opt-in like
// TestHelperMatchesPython: set STUDIO_XSD_HELPER to the built studio_xsd.exe).
func TestLongDataFolderValidates(t *testing.T) {
	helper := os.Getenv("STUDIO_XSD_HELPER")
	if helper == "" {
		t.Skip("set STUDIO_XSD_HELPER to the built studio_xsd.exe")
	}
	data := testutil.TempDir(t)
	for len(data) < 270 {
		data = filepath.Join(data, strings.Repeat("x", 40))
	}
	validator := New(Config{HelperPath: helper, SchemasDir: bundledSchemas, DataDir: data})
	defer validator.Close()
	source := []byte(`<?xml version="1.0"?><SCL xmlns="http://www.iec.ch/61850/2003/SCL" version="2007" ` +
		`revision="B" release="4"><Header id="x"/></SCL>`)
	verdict, err := validator.Validate(source, "SCL2007B4")
	if err != nil {
		t.Fatalf("data folder of %d characters: %v", len(data), err)
	}
	if verdict.Status != "valid" {
		t.Fatalf("verdict %+v", verdict)
	}
	t.Logf("data folder of %d characters validated", len(data))
}
