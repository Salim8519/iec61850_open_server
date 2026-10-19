package runtime

import (
	"path/filepath"
	"sort"
	"testing"
	"time"

	"studio-engine/internal/native"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

func percentile(values []float64, p float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	index := int(float64(len(sorted))*p) - 1
	if index < 0 {
		index = 0
	}
	return sorted[index]
}

// TestOperateTiming measures canvas CB operations like testdata/time_operate_python.py:
// "done" is the Operate call, "symbol" the first equipment_state with the new position.
func TestOperateTiming(t *testing.T) {
	requireMux(t, nativeDir(t))
	for _, keep := range []time.Duration{0, 10 * time.Second} {
		project := NewMemoryProject(loadProject(t, 21500+int64(keep/time.Second)*10))
		project.Current().Set("sources", []any{})
		bus := &MemoryBus{}
		c := New(Options{NativeDir: nativeDir(t), DataDir: testutil.TempDir(t), Bus: bus, Project: project, CommandKeep: keep})
		if err := c.Start(nil); err != nil {
			t.Fatal(err)
		}
		closed := false
		t.Cleanup(func() {
			if !closed {
				c.Close()
			}
		})
		time.Sleep(500 * time.Millisecond)
		var done, symbol []float64
		for i := 0; i < 30; i++ {
			closeIt := i%2 == 0
			wanted := int64(1)
			if closeIt {
				wanted = 2
			}
			cursor := len(bus.Events())
			started := time.Now()
			if _, err := c.Operate("cb1", closeIt, ""); err != nil {
				t.Fatal(err)
			}
			done = append(done, float64(time.Since(started).Microseconds())/1000)
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				found := false
				for _, event := range bus.Events()[cursor:] {
					if pyval.Str(event, "kind") == "equipment_state" && pyval.Str(event, "node_id") == "cb1" &&
						pyval.Equal(pyval.Get(event, "value"), wanted) {
						stamp, _ := pyval.Float(pyval.Get(event, "timestamp"))
						symbol = append(symbol, (stamp-float64(started.UnixNano())/1e9)*1000)
						found = true
						break
					}
				}
				if found {
					break
				}
				time.Sleep(time.Millisecond)
			}
			time.Sleep(100 * time.Millisecond)
		}
		c.Close()
		closed = true
		t.Logf("keep %v: done p50 %.1f p95 %.1f ms; symbol p50 %.1f p95 %.1f ms (%d ops)", keep,
			percentile(done, 0.5), percentile(done, 0.95), percentile(symbol, 0.5), percentile(symbol, 0.95), len(done))
	}
}

// requireMux skips when the native folder's studio_control lacks control_mux_v1, as in a
// checkout whose native binaries were built before the multiplexed canvas mode.
func requireMux(t *testing.T, folder string) {
	t.Helper()
	process, err := native.StartControl(filepath.Join(folder, "studio_control.exe"))
	if err != nil {
		t.Skipf("studio_control unavailable in %s: %v", folder, err)
	}
	defer process.Close()
	hello, err := process.Hello()
	if err != nil || !native.HasCapabilities(hello, []string{native.MuxCapability}) {
		t.Skipf("studio_control in %s is older than %s; rebuild the native tools", folder, native.MuxCapability)
	}
}
