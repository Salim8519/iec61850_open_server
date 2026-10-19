package service

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/store/archive"
	"studio-engine/internal/store/autosave"
	"studio-engine/internal/testutil"
)

// operate is one canvas operate save: a switch position written by the runtime.
func operate(t testing.TB, s *Service, n int) time.Duration {
	position := "open"
	if n%2 == 0 {
		position = "closed"
	}
	started := time.Now()
	if err := s.NodeData(fmt.Sprintf("node-%d", (n*37)%largeNodes), pyjson.NewObject().Set("initial_position", position)); err != nil {
		t.Fatal(err)
	}
	return time.Since(started)
}

func percentile(runs []time.Duration, p float64) time.Duration {
	sorted := append([]time.Duration(nil), runs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	index := int(float64(len(sorted)-1) * p)
	return sorted[index]
}

// BenchmarkNodeDataLargeProject is the click-path cost of a canvas operate save (journal
// line fsynced, snapshot coalesced in the background) in a 90k-point, 22 MB-SCL project.
// Target (speed plan S2): p95 <= 5 ms.
func BenchmarkNodeDataLargeProject(b *testing.B) {
	s := newLargeService(b)
	runs := make([]time.Duration, 0, b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		runs = append(runs, operate(b, s, i))
	}
	b.StopTimer()
	b.ReportMetric(float64(percentile(runs, 0.5).Microseconds())/1000, "p50-ms")
	b.ReportMetric(float64(percentile(runs, 0.95).Microseconds())/1000, "p95-ms")
}

// TestNodeDataCrashFolds: a copy of the data folder taken between operate saves (as after
// a crash: autosave plus journal, no Close) folds to exactly the in-memory project.
func TestNodeDataCrashFolds(t *testing.T) {
	s := newLargeService(t)
	s.snapshots.SetInterval(time.Hour)
	operate(t, s, 0)
	if err := s.snapshots.Flush(); err != nil {
		t.Fatal(err)
	}
	for round := 0; round < 3; round++ {
		for i := 0; i < 40; i++ {
			operate(t, s, round*40+i)
		}
		crashed := testutil.TempDir(t)
		// no snapshot runs during the copy (interval of an hour): the pair is consistent
		for _, name := range []string{autosave.AutosaveName, autosave.JournalName} {
			data, err := os.ReadFile(filepath.Join(s.config.DataDir, name))
			if err == nil {
				err = os.WriteFile(filepath.Join(crashed, name), data, 0o666)
			}
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
		}
		want, _ := pyjson.Dumps(s.current.Load().compact)
		applied, err := autosave.Fold(crashed, nil)
		if err != nil {
			t.Fatal(err)
		}
		payload, _ := os.ReadFile(filepath.Join(crashed, autosave.AutosaveName))
		stored, err := archive.Decode(payload)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := pyjson.Dumps(stored)
		t.Logf("round %d: %d journal records folded", round, applied)
		if string(got) != string(want) {
			t.Fatalf("round %d: folded copy differs from the project (%d vs %d bytes)", round, len(got), len(want))
		}
	}
}

// TestNodeDataLatencyLargeProject reports p50/p95/max over 200 operate saves and checks
// that the stored project still decodes to the in-memory one. Opt-in: STUDIO_TIMING=1.
func TestNodeDataLatencyLargeProject(t *testing.T) {
	if os.Getenv("STUDIO_TIMING") == "" {
		t.Skip("STUDIO_TIMING not set")
	}
	s := newLargeService(t)
	runs := make([]time.Duration, 0, 200)
	for i := 0; i < 200; i++ {
		runs = append(runs, operate(t, s, i))
	}
	t.Logf("NodeData over %d operates: p50 %v, p95 %v, max %v", len(runs),
		percentile(runs, 0.5), percentile(runs, 0.95), percentile(runs, 1))
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	checkStoredMatches(t, s)
}
