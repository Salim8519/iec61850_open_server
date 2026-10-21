package runtime

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// TestRunLogFormat checks the bytes log_writer.RunLog writes: runs.jsonl and manifest.jsonl
// lifecycle lines, and compact event records prefixed with run_id.
func TestRunLogFormat(t *testing.T) {
	root := testutil.TempDir(t)
	log, err := OpenRunLog(root)
	if err != nil {
		t.Fatal(err)
	}
	log.Write(pyval.Obj("id", int64(1), "kind", "runtime", "timestamp", 1.5, "state", "running", "running", []any{"a"}))
	log.Write(pyval.Obj("id", int64(2), "kind", "signal", "timestamp", 2.0, "ied_id", "a", "value", 1.0, "note", "é"))
	log.Close()
	events, _ := os.ReadFile(filepath.Join(root, log.RunID, "events-0000.jsonl"))
	want := `{"run_id":"` + log.RunID + `","id":1,"kind":"runtime","timestamp":1.5,"state":"running","running":["a"]}` + "\n" +
		`{"run_id":"` + log.RunID + `","id":2,"kind":"signal","timestamp":2.0,"ied_id":"a","value":1.0,"note":"\u00e9"}` + "\n"
	if string(events) != want {
		t.Fatalf("events:\n%s\nwant:\n%s", events, want)
	}
	stamp := `\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(\.\d{6})?\+00:00`
	manifest, _ := os.ReadFile(filepath.Join(root, log.RunID, "manifest.jsonl"))
	pattern := regexp.MustCompile(`^\{"kind":"run_started","run_id":"` + log.RunID + `","at":"` + stamp + `","application":"0\.1\.0"\}\n` +
		`\{"kind":"segment_opened","run_id":"` + log.RunID + `","at":"` + stamp + `","file":"events-0000\.jsonl"\}\n` +
		`\{"kind":"run_closed","run_id":"` + log.RunID + `","at":"` + stamp + `"\}\n$`)
	if !pattern.Match(manifest) {
		t.Fatalf("manifest:\n%s", manifest)
	}
	runs, _ := os.ReadFile(filepath.Join(root, "runs.jsonl"))
	if lines := strings.Count(string(runs), "\n"); lines != 2 {
		t.Fatalf("runs.jsonl has %d lines", lines)
	}
}
