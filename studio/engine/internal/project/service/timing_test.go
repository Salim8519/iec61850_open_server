package service

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"studio-engine/internal/testutil"
	"testing"
	"time"

	"studio-engine/internal/scl"
	"studio-engine/internal/store/modelcache"
)

func median(runs []time.Duration) time.Duration {
	sorted := append([]time.Duration(nil), runs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	return sorted[len(sorted)/2]
}

// TestOpenSaveTimings compares open (restore) and save of the same large Python-built
// project in both engines. Opt-in: STUDIO_TIMING=1 and STUDIO_PYTHON.
func TestOpenSaveTimings(t *testing.T) {
	if os.Getenv("STUDIO_TIMING") == "" {
		t.Skip("STUDIO_TIMING not set")
	}
	archive := filepath.Join(testutil.TempDir(t), "big.subsim")
	python(t, "big", archive)
	payload, err := os.ReadFile(archive)
	if err != nil {
		t.Fatal(err)
	}
	dir := testutil.TempDir(t)
	s, err := New(Config{DataDir: dir, ModelCache: modelcache.New(dir), Importer: &scl.Importer{Validator: loadSeed(t)}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	restores, saves := []time.Duration{}, []time.Duration{}
	for i := 0; i < 5; i++ {
		started := time.Now()
		if err := s.Restore(payload, false); err != nil {
			t.Fatal(err)
		}
		restores = append(restores, time.Since(started))
	}
	for i := 0; i < 5; i++ {
		started := time.Now()
		if err := s.Save(); err != nil {
			t.Fatal(err)
		}
		saves = append(saves, time.Since(started))
	}
	fields := strings.Fields(python(t, "savetime", archive))
	t.Logf("archive %d bytes, %d IEDs", len(payload), len(iedList(s.Project())))
	t.Logf("open:       Go %v rebuild, %v from model cache  Python %ss", restores[0], median(restores[1:]), fields[0])
	t.Logf("first save: Go %v  Python %ss", saves[0], fields[1])
	t.Logf("save:       Go %v  Python %ss", median(saves), fields[2])
}
