package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCappedLogNeverExceedsLimit(t *testing.T) {
	dir := t.TempDir()
	log, err := NewForTest(dir, 1024, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 500; i++ {
		log.Event("request", map[string]any{"path": strings.Repeat("x", 40), "n": i})
	}
	log.Close()
	entries, _ := os.ReadDir(dir)
	if len(entries) != 3 {
		t.Fatalf("expected 3 rotated files, got %d", len(entries))
	}
	var total int64
	for _, entry := range entries {
		info, _ := entry.Info()
		total += info.Size()
		if info.Size() > 1024 {
			t.Fatalf("%s is %d bytes, above the 1024 cap", entry.Name(), info.Size())
		}
	}
	if total > 3*1024 {
		t.Fatalf("total log size %d exceeds cap", total)
	}
	newest, _ := os.ReadFile(filepath.Join(dir, "server.log"))
	if !strings.Contains(string(newest), `"n":499`) {
		t.Fatal("newest event not in the current file")
	}
}

func TestMaskKeyHidesAllButLastGroup(t *testing.T) {
	if got := MaskKey("SUBS-ABCD-EFGH-JKMN-PQRS"); got != "SUBS-****-****-****-PQRS" {
		t.Fatalf("got %s", got)
	}
}
