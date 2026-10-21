package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/project"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/testutil"
)

var demoSCD = filepath.Join("..", "..", "..", "..", "scd", "open_substation.scd")

// demoStation imports the public demo station (4 IEDs, GOOSE and SV) and moves IED i to
// prefix.(11+i), port base+1+i.
func demoStation(t testing.TB, engine *Engine, prefix string, base int) {
	t.Helper()
	source, err := os.ReadFile(demoSCD)
	if err != nil {
		t.Fatal(err)
	}
	models, err := engine.Projects.AddSCL(source, filepath.Base(demoSCD), scl.SelectAll)
	if err != nil {
		t.Fatal(err)
	}
	for index, model := range models {
		id, _ := model.Get("id")
		address, port := prefix+"."+itoa(11+index), int64(base+1+index)
		if _, err := engine.Projects.Update(func(draft *pyjson.Object) (*pyjson.Object, error) {
			ied, err := project.FindIED(draft, id.(string))
			if err == nil {
				ied.Set("ip", address).Set("port", port)
			}
			return ied, err
		}); err != nil {
			t.Fatal(err)
		}
	}
}

// TestComposedStopTiming (STUDIO_TIMING=1, real workers, STUDIO_PYTHON): Run and Stop of the
// demo station through the composed HTTP routes, next to the frozen Python Coordinator on
// the same station (gap 2: Stop was 286 ms in Go vs 159 ms in Python in the UI).
func TestComposedStopTiming(t *testing.T) {
	if os.Getenv("STUDIO_TIMING") != "1" {
		t.Skip("set STUDIO_TIMING=1 to measure")
	}
	python := os.Getenv("STUDIO_PYTHON")
	nativeDir := muxNative(t)
	if python == "" {
		t.Skip("set STUDIO_PYTHON to time the Python side")
	}
	const rounds, prefix = 12, "127.67.3"
	hosts := []string{prefix + ".11", prefix + ".12", prefix + ".13", prefix + ".14"}
	base := freePortBase(t, 4, hosts...)
	out := filepath.Join(testutil.TempDir(t), "python.json")
	if output, err := exec.Command(python, "-I", filepath.Join("testdata", "stop_timing_python.py"), nativeDir,
		demoSCD, prefix, itoa(base), itoa(rounds), out).CombinedOutput(); err != nil {
		t.Fatalf("stop_timing_python.py: %v %s", err, output)
	}
	raw, _ := os.ReadFile(out)
	decoded, _ := pyjson.Decode(raw)
	seconds := func(key string) []time.Duration {
		var result []time.Duration
		for _, value := range field(decoded, key).([]any) {
			result = append(result, time.Duration(value.(float64)*float64(time.Second)))
		}
		return result
	}
	engine := composeAt(t, testutil.TempDir(t), nativeDir, pythonVerdicts(t, demoSCD))
	defer engine.Close()
	demoStation(t, engine, prefix, base)
	var starts, stops []time.Duration
	for i := 0; i < rounds; i++ {
		started := time.Now()
		mustCall(t, engine, "POST", "/api/v1/runtime/start", nil)
		starts = append(starts, time.Since(started))
		time.Sleep(500 * time.Millisecond)
		started = time.Now()
		mustCall(t, engine, "POST", "/api/v1/runtime/stop", nil)
		stops = append(stops, time.Since(started))
		time.Sleep(200 * time.Millisecond)
	}
	report := func(name string, values []time.Duration) {
		p50, p95 := percentiles(values)
		t.Logf("%-10s p50 %6.1f ms  max %6.1f ms  %v", name, p50, p95, values)
	}
	report("py start", seconds("start"))
	report("go start", starts)
	report("py stop", seconds("stop"))
	report("go stop", stops)
}
