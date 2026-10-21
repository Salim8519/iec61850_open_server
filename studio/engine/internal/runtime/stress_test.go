package runtime

import (
	"bytes"
	"math/rand"
	"os"
	goruntime "runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// countingBus builds the event envelope like MemoryBus but keeps only a count, so a long
// stress run does not grow memory.
type countingBus struct {
	mu    sync.Mutex
	count int64
	kinds map[string]int64
}

func (b *countingBus) Publish(kind string, payload *pyjson.Object) *pyjson.Object {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.count++
	if b.kinds == nil {
		b.kinds = map[string]int64{}
	}
	b.kinds[kind]++
	event := pyval.Obj("id", b.count, "kind", kind, "timestamp", float64(time.Now().UnixNano())/1e9)
	for _, key := range payload.Keys() {
		value, _ := payload.Get(key)
		event.Set(key, value)
	}
	return event
}

// stressLane is one goroutine of the stress mix with a progress counter for the watchdog.
type stressLane struct {
	name  string
	ops   atomic.Int64
	since atomic.Int64 // unix nanos of the last finished operation
	run   func(r *rand.Rand)
}

// TestLockOrderStress runs Run/Stop/Pause, canvas operations with native feedback, manual
// writes, pulses, source and project edits under the cycle lock, live values, hub
// subscribers that call back into the runtime, and readers, all at once on fake workers.
// Any lane without progress for 10 s dumps every goroutine and fails (lock order:
// lifeMu -> commandMu -> project mutation -> table.mu -> emitMu/bus.mu). STUDIO_STRESS=1
// enables it; STUDIO_STRESS_SECONDS sets the length (60 s).
func TestLockOrderStress(t *testing.T) {
	if os.Getenv("STUDIO_STRESS") != "1" {
		t.Skip("set STUDIO_STRESS=1 to run the 60 s lock-order stress test")
	}
	seconds := 60
	if text := os.Getenv("STUDIO_STRESS_SECONDS"); text != "" {
		seconds, _ = strconv.Atoi(text)
	}
	folder := fakeNativeDir(t)
	project, a, b := fakeProject(t, 3, 22300, true)
	store := NewMemoryProject(project)
	bus := &countingBus{}
	c := New(Options{NativeDir: folder, DataDir: testutil.TempDir(t), Bus: bus, Project: store})
	defer c.Close()
	var notified atomic.Int64
	cancelMain := c.Subscribe(func(change Change) {
		notified.Add(1)
		if change.Ref != "" {
			c.Record(change.IEDID, change.Ref) // subscribers read back into the runtime
		}
		c.RunState()
	})
	defer cancelMain()
	analog, loc := "BayACTRL/GGIO1.AnIn2.mag.f", "BayACTRL/XCBR1.Loc.stVal"
	lanes := []*stressLane{
		{name: "lifecycle", run: func(r *rand.Rand) {
			_ = c.Start(nil)
			time.Sleep(time.Duration(50+r.Intn(250)) * time.Millisecond)
			switch r.Intn(4) {
			case 0:
				_ = c.Pause()
				_ = c.Pause()
			case 1:
				_ = c.Stop([]string{b})
				_ = c.Start([]string{b})
			}
			time.Sleep(time.Duration(r.Intn(200)) * time.Millisecond)
			_ = c.Stop(nil)
			time.Sleep(time.Duration(r.Intn(100)) * time.Millisecond) // edits need stopped windows
		}},
		{name: "operate", run: func(r *rand.Rand) {
			_, _ = c.Operate("cb1", r.Intn(2) == 0, "")
			_, _ = c.Operate("ds1", nil, []string{"open", "close"}[r.Intn(2)])
			_, _ = c.Operate("pb1", nil, "press")
			time.Sleep(time.Millisecond)
		}},
		{name: "writes", run: func(r *rand.Rand) {
			_, _ = c.Update(a, analog, r.Float64()*100, nil, "manual")
			_, _ = c.Pulse(a, loc, true, int64(20+r.Intn(100)))
			_ = c.ApplySamples(a, []RefValue{{analog, r.Float64()}}, "goose_input")
			_ = c.ReleaseOverride(a, "BayACTRL/GGIO1.AnIn1.mag.f")
			time.Sleep(time.Millisecond)
		}},
		{name: "edits", run: func(r *rand.Rand) {
			_ = c.Stopped(func() error {
				_ = store.NodeData("cb1", pyval.Obj("initial_position", []string{"open", "closed"}[r.Intn(2)]))
				c.Emit("ieds_removed", pyval.Obj("ied_ids", []any{}, "count", int64(0)))
				c.ResetValues(b)
				return nil
			})
			lock := c.CycleLock()
			lock.Lock()
			_ = store.NodeData("ds1", pyval.Obj("initial_position", "open"))
			lock.Unlock()
			_ = c.ConfigureSource(a, "BayACTRL/GGIO1.AnIn1.mag.f", pyval.Obj("mode", "ramp", "offset", 1.5,
				"amplitude", 10.0, "period", 2.0, "seed", int64(r.Intn(9))))
			_ = c.StartScenario("s1")
			time.Sleep(time.Duration(r.Intn(20)) * time.Millisecond)
		}},
		{name: "live", run: func(r *rand.Rand) {
			c.Values().SetLive(b, []RefValue{{"BayBCTRL/GGIO1.AnIn2.mag.f", r.Float64()}})
			if r.Intn(10) == 0 {
				c.Values().MarkStale(b)
				c.SetIEDState(b, "running")
			}
			c.PublishChanged([]PointKey{{a, "BayACTRL/XCBR1.Pos.stVal"}})
			time.Sleep(time.Millisecond)
		}},
		{name: "subscribers", run: func(r *rand.Rand) {
			cancel := c.Subscribe(func(change Change) { c.Signal(change.IEDID, change.Ref) })
			time.Sleep(time.Duration(r.Intn(30)) * time.Millisecond)
			cancel()
		}},
		{name: "readers", run: func(r *rand.Rand) {
			_, _ = c.Signal(a, analog)
			c.Source(a, "BayACTRL/GGIO1.AnIn1.mag.f")
			c.IedStates()
			c.RunningSequences()
			_, _, _ = c.Endpoint(a)
			c.EquipmentStates()
			c.MmsAccess()
			c.Values().Get(b, "BayBCTRL/GGIO1.AnIn2.mag.f")
			time.Sleep(time.Millisecond)
		}},
	}
	stop := make(chan struct{})
	var group sync.WaitGroup
	for i, lane := range lanes {
		lane.since.Store(time.Now().UnixNano())
		group.Add(1)
		go func(seed int64, lane *stressLane) {
			defer group.Done()
			r := rand.New(rand.NewSource(seed))
			for {
				select {
				case <-stop:
					return
				default:
				}
				lane.run(r)
				lane.ops.Add(1)
				lane.since.Store(time.Now().UnixNano())
			}
		}(int64(i+1), lane)
	}
	stalled := watchLanes(t, lanes, time.Duration(seconds)*time.Second)
	close(stop)
	finished := make(chan struct{})
	go func() { group.Wait(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(30 * time.Second):
		dumpGoroutines(t)
		t.Fatal("lanes did not finish within 30 s of the end")
	}
	if stalled != "" {
		t.Fatalf("lane %s stalled for more than 10 s", stalled)
	}
	for _, lane := range lanes {
		t.Logf("%-12s %d operations", lane.name, lane.ops.Load())
	}
	t.Logf("events %d, hub notifications delivered %d, goroutines %d", bus.count, notified.Load(),
		goruntime.NumGoroutine())
	for _, kind := range []string{"runtime", "signal", "equipment_feedback_applied", "control_applied",
		"equipment_feedback_observed", "pulse_completed", "scenario_started", "source", "ieds_removed"} {
		t.Logf("  %-28s %d", kind, bus.kinds[kind])
	}
}

// watchLanes returns the first lane without progress for 10 s ("" after duration).
func watchLanes(t *testing.T, lanes []*stressLane, duration time.Duration) string {
	end := time.Now().Add(duration)
	for time.Now().Before(end) {
		time.Sleep(500 * time.Millisecond)
		for _, lane := range lanes {
			if time.Since(time.Unix(0, lane.since.Load())) > 10*time.Second {
				dumpGoroutines(t)
				return lane.name
			}
		}
	}
	return ""
}

func dumpGoroutines(t *testing.T) {
	var buffer bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&buffer, 2)
	t.Log(buffer.String())
}
