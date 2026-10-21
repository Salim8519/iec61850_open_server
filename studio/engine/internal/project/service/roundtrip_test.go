package service

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"studio-engine/internal/testutil"
	"testing"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

const worktree = "../../../../.."

// seedValidator answers with the XSD verdicts CPython recorded for the public corpus.
type seedValidator map[string]scl.SchemaVerdict

func (v seedValidator) Validate(source []byte, schema string) (scl.SchemaVerdict, error) {
	sum := sha256.Sum256(source)
	if verdict, ok := v[hex.EncodeToString(sum[:])+"/"+schema]; ok {
		return verdict, nil
	}
	return scl.SchemaVerdict{}, apperr.RuntimeError("no recorded XSD verdict")
}

func loadSeed(t testing.TB) seedValidator {
	data, err := os.ReadFile("../../scl/testdata/golden/xsd_seed.json")
	if err != nil {
		t.Skip("xsd seed missing")
	}
	decoded, _ := pyjson.Decode(data)
	seed := seedValidator{}
	for _, raw := range decoded.([]any) {
		item := raw.(*pyjson.Object)
		verdict := scl.SchemaVerdict{Status: str(item, "status")}
		list, _ := item.Get("errors")
		for _, e := range list.([]any) {
			entry := e.(*pyjson.Object)
			line, _ := entry.Get("line")
			verdict.Errors = append(verdict.Errors, scl.SchemaError{Line: int(line.(int64)), Message: str(entry, "message")})
		}
		seed[str(item, "sha256")+"/"+str(item, "schema")] = verdict
	}
	return seed
}

func str(o *pyjson.Object, key string) string {
	value, _ := o.Get(key)
	text, _ := value.(string)
	return text
}

func newService(t testing.TB) *Service {
	s, err := New(Config{DataDir: testutil.TempDir(t), Importer: &scl.Importer{Validator: loadSeed(t)}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// publicSnapshot is the snapshot without the per-process session id and revision.
func publicSnapshot(t testing.TB, s *Service) string {
	snapshot := s.Snapshot()
	snapshot.Delete("session_id")
	snapshot.Delete("revision")
	data, err := pyjson.Dumps(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func python(t *testing.T, args ...string) string {
	interpreter := os.Getenv("STUDIO_PYTHON")
	if interpreter == "" {
		t.Skip("STUDIO_PYTHON not set")
	}
	command := exec.Command(interpreter, append([]string{"-I", filepath.Join("testdata", "roundtrip.py")}, args...)...)
	out, err := command.Output()
	if err != nil {
		t.Fatalf("python %v: %v %s", args, err, out)
	}
	return strings.TrimRight(string(out), "\r\n")
}

// TestPythonArchiveOpensInGo: a project built and saved by the Python engine restores in Go
// to the same snapshot, and Go's re-export reopens in Python to the same snapshot again.
func TestPythonArchiveRoundTrip(t *testing.T) {
	folder := testutil.TempDir(t)
	pythonFile := filepath.Join(folder, "python.subsim")
	want := python(t, "make", pythonFile)
	payload, err := os.ReadFile(pythonFile)
	if err != nil {
		t.Fatal(err)
	}
	s := newService(t)
	if err := s.Restore(payload, true); err != nil {
		t.Fatal(err)
	}
	if got := publicSnapshot(t, s); got != want {
		t.Fatalf("Go restore of a Python archive differs:\n got  %.400s\n want %.400s", got, want)
	}
	exported, err := s.Export()
	if err != nil {
		t.Fatal(err)
	}
	goFile := filepath.Join(folder, "go.subsim")
	if err := os.WriteFile(goFile, exported, 0o600); err != nil {
		t.Fatal(err)
	}
	if again := python(t, "open", goFile); again != want {
		t.Fatalf("Python restore of the Go archive differs:\n got  %.400s\n want %.400s", again, want)
	}
	started := time.Now()
	const runs = 5
	for i := 0; i < runs; i++ {
		if err := newService(t).Restore(payload, false); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("open (restore) Go %.1f ms median-ish over %d, Python %s s", float64(time.Since(started).Microseconds())/1000/runs,
		runs, python(t, "time", pythonFile))
	_ = worktree
}
