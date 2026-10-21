package app

import (
	"os"
	"sort"
	"testing"
	"time"

	"studio-engine/internal/testutil"
)

func percentiles(values []time.Duration) (float64, float64) {
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	at := func(p float64) float64 {
		index := int(float64(len(sorted))*p+0.5) - 1
		if index < 0 {
			index = 0
		}
		return ms(sorted[index])
	}
	return at(0.5), at(0.95)
}

// TestComposedOperateTiming (STUDIO_TIMING=1, real workers): canvas CB operate through the
// composed HTTP handler — "answer" is the POST, "symbol" the equipment_state event's own
// timestamp — and the diagram save (PUT /project/diagram) while stopped.
func TestComposedOperateTiming(t *testing.T) {
	if os.Getenv("STUDIO_TIMING") != "1" {
		t.Skip("set STUDIO_TIMING=1 to measure")
	}
	engine := composeAt(t, testutil.TempDir(t), muxNative(t))
	defer engine.Close()
	clickProject(t, engine)
	var saves []time.Duration
	for i := 0; i < 30; i++ {
		snapshot := mustCall(t, engine, "GET", "/api/v1/project", nil)
		diagram := field(snapshot, "diagram")
		started := time.Now()
		mustCall(t, engine, "PUT", "/api/v1/project/diagram",
			obj("diagram", diagram, "revision", field(snapshot, "revision"), "tab_id", "tab-1"))
		saves = append(saves, time.Since(started))
	}
	mustCall(t, engine, "POST", "/api/v1/runtime/start", nil)
	time.Sleep(500 * time.Millisecond)
	operateCB(t, engine, false) // starts the warm studio_control (first click after start)
	var answers, symbols []time.Duration
	for i := 0; i < 40; i++ {
		answer, symbol := operateCB(t, engine, i%2 == 0)
		answers, symbols = append(answers, answer), append(symbols, symbol)
		time.Sleep(50 * time.Millisecond)
	}
	mustCall(t, engine, "POST", "/api/v1/runtime/stop", nil)
	a50, a95 := percentiles(answers)
	s50, s95 := percentiles(symbols)
	d50, d95 := percentiles(saves)
	t.Logf("operate answer p50 %.1f p95 %.1f ms; symbol p50 %.1f p95 %.1f ms (%d ops); diagram save p50 %.1f p95 %.1f ms",
		a50, a95, s50, s95, len(answers), d50, d95)
}
