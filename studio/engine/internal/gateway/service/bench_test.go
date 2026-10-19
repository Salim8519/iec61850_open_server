package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/nativetest"
	"studio-engine/internal/tools/pyoracle"
)

// benchRef mirrors testdata/bench_gateway.py: odd points are floats, even points singles.
func benchRef(number int) (string, string, any) {
	if number%2 == 1 {
		return fmt.Sprintf("BayCTRL/GGIO1.AnIn%d.mag.f", number), "float", 1.5
	}
	return fmt.Sprintf("BayCTRL/GGIO1.Ind%d.stVal", number), "single", false
}

// runGoBench runs the Go gateway with count points while rate values change per second.
func runGoBench(t *testing.T, count, rate int, window time.Duration, port int64) (time.Duration, time.Duration) {
	executable := nativetest.Program(t, "studio_gateway.exe")
	hub := newHub()
	var signals, points []any
	for number := 0; number < count; number++ {
		ref, kind, value := benchRef(number)
		base := ref[:len(ref)-len(".mag.f")]
		if kind == "single" {
			base = ref[:len(ref)-len(".stVal")]
		}
		valueKind, fc := "BOOLEAN", "ST"
		if kind == "float" {
			valueKind, fc = "FLOAT32", "MX"
		}
		hub.define("bay", ref, Signal{Value: value, QualityRef: base + ".q", TimestampRef: base + ".t"})
		hub.define("bay", base+".q", Signal{Value: int64(0)})
		hub.define("bay", base+".t", Signal{Value: int64(1700000000000)})
		signals = append(signals, pyjson.NewObject().Set("ref", ref).Set("fc", fc).Set("kind", valueKind))
		points = append(points, pyjson.NewObject().Set("kind", kind).Set("ied", "bay").Set("ref", ref).
			Set("address", int64(number+1)).Set("scale", 1.0).Set("description", ""))
	}
	ied := pyjson.NewObject().Set("id", "bay").Set("name", "Bay").Set("role", "simulated").Set("state", "running").
		Set("signals", signals).Set("controls", []any{})
	section := gateway.EmptyGateway()
	get(section, "settings").(*pyjson.Object).Set("iec104", get(get(section, "settings").(*pyjson.Object), "iec104").(*pyjson.Object).Set("port", port))
	get(section, "points").(*pyjson.Object).Set("iec104", points)
	manager := NewManager(&fakeProjects{ieds: []*pyjson.Object{ied}, section: section}, hub, nil, executable, nil)
	if _, err := manager.Start(); err != nil {
		t.Fatal(err)
	}
	defer manager.Stop(false)
	return measureChanges(hub, count, rate, window)
}

// runHarnessBaseline is the CPU of the fake hub and the value changer alone (no gateway), so
// the test's own load can be told apart from the gateway's.
func runHarnessBaseline(count, rate int, window time.Duration) (time.Duration, time.Duration) {
	hub := newHub()
	for number := 0; number < count; number++ {
		ref, _, value := benchRef(number)
		hub.define("bay", ref, Signal{Value: value})
	}
	cancel := hub.Subscribe(func(Change) {})
	defer cancel()
	return measureChanges(hub, count, rate, window)
}

// measureChanges changes rate values per second (in 10 ms steps) and returns the process CPU
// used during window, after a 2 s settling time.
func measureChanges(hub *fakeHub, count, rate int, window time.Duration) (time.Duration, time.Duration) {
	stop := make(chan struct{})
	go func() {
		step, number := 0, 0
		for rate > 0 {
			for i := 0; i < max(1, rate/100); i++ {
				ref, kind, _ := benchRef(number % count)
				var value any = step%2 == 1
				if kind == "float" {
					value = float64(step%100) / 10
				}
				hub.set("bay", ref, value)
				number++
			}
			step++
			select {
			case <-stop:
				return
			case <-time.After(10 * time.Millisecond):
			}
		}
	}()
	time.Sleep(2 * time.Second)
	before, beforeTicks := processCPU(), processTicks()
	time.Sleep(window)
	cpu, ticks := processCPU()-before, processTicks()-beforeTicks
	close(stop)
	return cpu, ticks
}

// TestGatewayCPUAgainstPython measures backend CPU for 5000 points (set STUDIO_GATEWAY_BENCH=1).
func TestGatewayCPUAgainstPython(t *testing.T) {
	if os.Getenv("STUDIO_GATEWAY_BENCH") == "" {
		t.Skip("set STUDIO_GATEWAY_BENCH=1 to measure gateway CPU (about 1 minute)")
	}
	t.Setenv("STUDIO_NATIVE_BIN", nativetest.Dir())
	window := 20 * time.Second
	for index, rate := range []int{0, 1000} {
		port := int64(20470 + index)
		python := []any{"not measured (STUDIO_GATEWAY_BENCH=go)"}
		if os.Getenv("STUDIO_GATEWAY_BENCH") != "go" {
			python = pyoracle.Run(t, filepath.Join("testdata", "bench_gateway.py"), nil, "5000", strconv.Itoa(rate),
				strconv.Itoa(int(window.Seconds())), strconv.FormatInt(port, 10))
		}
		total, totalTicks := runGoBench(t, 5000, rate, window, port+10)
		harness, harnessTicks := runHarnessBaseline(5000, rate, window)
		cpu := max(0, total-harness)
		goPercent := 100 * cpu.Seconds() / window.Seconds()
		tickPercent := 100 * max(0, totalTicks-harnessTicks).Seconds() / window.Seconds()
		encoded, _ := pyjson.Dumps(python[0])
		t.Logf("rate %d/s: python %s; go process %.3f s, harness alone %.3f s, gateway %.3f s = %.2f %% of one core"+
			" (tick-based process times: %.2f %%)", rate, encoded, total.Seconds(), harness.Seconds(), cpu.Seconds(),
			goPercent, tickPercent)
		if goPercent > 1 {
			t.Errorf("Go gateway uses %.2f %% of a core with 5000 points at %d changes/s (budget 1 %%)", goPercent, rate)
		}
	}
}
