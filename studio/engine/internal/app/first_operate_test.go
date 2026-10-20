package app

import (
	"os"
	"testing"
	"time"

	"studio-engine/internal/testutil"
)

// TestComposedFirstOperateTiming (STUDIO_TIMING=1, real workers): the first canvas CB
// operate after Run (gap 3: it paid the studio_control start, 0.1-0.5 s) next to a warm one.
// Each round composes a fresh engine, so studio_control has never run in it.
func TestComposedFirstOperateTiming(t *testing.T) {
	if os.Getenv("STUDIO_TIMING") != "1" {
		t.Skip("set STUDIO_TIMING=1 to measure")
	}
	nativeDir := muxNative(t)
	var firsts, warms []time.Duration
	for round := 0; round < 6; round++ {
		engine := composeAt(t, testutil.TempDir(t), nativeDir)
		clickProject(t, engine)
		mustCall(t, engine, "POST", "/api/v1/runtime/start", nil)
		time.Sleep(250 * time.Millisecond) // the UI shows "Running", then the user clicks
		first, _ := operateCB(t, engine, true)
		time.Sleep(50 * time.Millisecond)
		warm, _ := operateCB(t, engine, false)
		firsts, warms = append(firsts, first), append(warms, warm)
		mustCall(t, engine, "POST", "/api/v1/runtime/stop", nil)
		if err := engine.Close(); err != nil {
			t.Fatal(err)
		}
	}
	f50, f95 := percentiles(firsts)
	w50, w95 := percentiles(warms)
	t.Logf("first operate after Run p50 %.1f max %.1f ms %v; next operate p50 %.1f max %.1f ms", f50, f95, firsts, w50, w95)
}
