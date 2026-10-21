package selftest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"studio-engine/internal/testutil"
)

// pythonFixtureSHA256 is sha256(self_test.fixture()) from Python 3.12's ElementTree
// (python -I testdata/python_check.py fixture studio/backend OUT); it is also the
// first verdict of testdata/xsd_seed.json.
const pythonFixtureSHA256 = "cdffd0003dc4a28c7d73fcdd633a6213db7d1e4e86e161d11a48be018c012690"

func TestFixtureIsPythonBytes(t *testing.T) {
	source, err := Fixture()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(source)
	if got := hex.EncodeToString(digest[:]); got != pythonFixtureSHA256 {
		t.Fatalf("fixture sha256 %s, Python wrote %s", got, pythonFixtureSHA256)
	}
	for _, part := range [][]byte{[]byte(`<ns0:SCL xmlns:ns0="http://www.iec.ch/61850/2003/SCL" version="2007"`),
		[]byte(`<ns0:DOType id="SelfTestArray" cdc="MV"><ns0:SDO name="samples" type="StudioMV" count="3" /></ns0:DOType><ns0:DAType`),
		[]byte(`<ns0:DO name="AnIn2" type="SelfTestArray" />`)} {
		if !bytes.Contains(source, part) {
			t.Errorf("fixture lacks %s", part)
		}
	}
	python := os.Getenv("STUDIO_PYTHON")
	if python == "" {
		return
	}
	out := filepath.Join(testutil.TempDir(t), "fixture.xml")
	backend, _ := filepath.Abs(filepath.Join("..", "..", "..", "backend"))
	if output, err := exec.Command(python, "-I", filepath.Join("testdata", "python_check.py"), "fixture",
		backend, out).CombinedOutput(); err != nil {
		t.Fatalf("python_check.py fixture: %v %s", err, output)
	}
	expected, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(source, expected) {
		t.Fatalf("fixture differs from Python's bytes (%d vs %d bytes)", len(source), len(expected))
	}
}

func TestWithinIsLexicalAndCaseInsensitive(t *testing.T) {
	for _, item := range []struct {
		path, folder string
		want         bool
	}{
		{`C:\Data\studio-check-1\recovery\a.subsim`, `C:\Data\studio-check-1`, true},
		{`c:\data\STUDIO-CHECK-1`, `C:\Data\studio-check-1`, true},
		{`C:\Data\studio-check-10\a`, `C:\Data\studio-check-1`, false},
		{`C:\Data\other\..\studio-check-1x`, `C:\Data\studio-check-1`, false},
		{``, `C:\Data`, false},
	} {
		if filepath.Separator != '\\' {
			t.Skip("Windows paths")
		}
		if got := within(item.path, item.folder); got != item.want {
			t.Errorf("within(%q, %q) = %v, want %v", item.path, item.folder, got, item.want)
		}
	}
}
