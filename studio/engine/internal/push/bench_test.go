package push

import (
	"fmt"
	"runtime"
	"runtime/metrics"
	"strings"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

// goCPU is the Go runtime's estimate of the process CPU time spent so far.
func goCPU() time.Duration {
	sample := []metrics.Sample{{Name: "/cpu/classes/total:cpu-seconds"}, {Name: "/cpu/classes/idle:cpu-seconds"}}
	metrics.Read(sample)
	return time.Duration((sample[0].Value.Float64() - sample[1].Value.Float64()) * float64(time.Second))
}

// TestIdleTabsCostNothing: 10 tabs with open streams and subscribed topics, nothing
// changing: no allocation and no measurable CPU (no tickers; streams wait on channels).
func TestIdleTabsCostNothing(t *testing.T) {
	route := newBody(`{"a":1}`)
	area, bus := testArea(t, WallClock, &Topic{Name: "health", Get: route.get, Every: time.Second,
		While: func() bool { return false }, Triggers: []string{"live_*"}},
		&Topic{Name: "gateway.points", Get: route.get, Values: AnyValue})
	srv := server(t, area, bus)
	for tab := 0; tab < 10; tab++ {
		id := fmt.Sprintf("tab-%d", tab)
		call(t, "PUT", srv.URL+"/api/v1/push/topics", `{"client":"`+id+`","topics":[{"t":"health"},{"t":"gateway.points"}]}`)
		stream := openStream(t, srv.URL+"/api/v1/push?client="+id)
		stream.wait(t, "full frames", func(s string) bool { return strings.Count(s, `"full"`) == 2 })
	}
	time.Sleep(300 * time.Millisecond)
	first, _ := idleWindow()
	allocs, used := idleWindow()
	t.Logf("first window %d allocations (runtime warm-up after GC), second window %d", first, allocs)
	t.Logf("10 idle tabs, 2 s: %d allocations, Go CPU %v (%.3f %% of one core)", allocs, used,
		100*used.Seconds()/2)
	if allocs > 4 || used > 40*time.Millisecond { // the measurement allocates 2
		t.Fatalf("idle tabs cost %d allocations and %v CPU", allocs, used)
	}
}

// idleWindow counts allocations and Go CPU over 2 s of doing nothing.
func idleWindow() (uint64, time.Duration) {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	cpu := goCPU()
	time.Sleep(2 * time.Second)
	used := goCPU() - cpu
	runtime.ReadMemStats(&after)
	return after.Mallocs - before.Mallocs, used
}

// pointsBody is a gateway point table of n rows (Flask style), row changed carrying value.
func pointsBody(n, changed int, value string) []byte {
	var out strings.Builder
	out.WriteString(`{"points":[`)
	for index := 0; index < n; index++ {
		if index > 0 {
			out.WriteByte(',')
		}
		text := "1.5"
		if index == changed {
			text = value
		}
		fmt.Fprintf(&out, `{"address":%d,"description":"","id":"bay:GGIO1.AnIn%d.mag.f:float","ied":"bay",`+
			`"ied_name":"Bay","ied_role":"simulated","kind":"float","monitor_only":false,"online":true,`+
			`"ref":"BayCTRL/GGIO1.AnIn%d.mag.f","scale":1.0,"value_text":"%s"}`, index+1, index, index, text)
	}
	out.WriteString("]}\n")
	return []byte(out.String())
}

// BenchmarkKeyedBuild5000: one build of a 5000-point table where one row changed (parse,
// compare, delta frame) — the cost the push adds per coalesced change of gateway.points.
func BenchmarkKeyedBuild5000(b *testing.B) {
	bodies := [][]byte{pointsBody(5000, 17, "2.5"), pointsBody(5000, 17, "3.5")}
	turn := 0
	topic := &Topic{Name: "gateway.points", Kind: Keyed, List: "points", Key: "id",
		Get: func(string) (int, []byte) { turn++; return 200, bodies[turn%2] }}
	area := New(Options{Topics: []*Topic{topic}})
	defer area.Close()
	in := newInstance(area, topic, "")
	in.build()
	b.ReportAllocs()
	b.SetBytes(int64(len(bodies[0])))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		in.throttle.running = true
		in.build()
	}
	b.StopTimer()
	if !strings.Contains(string(in.full), "") || in.rev < 2 {
		b.Fatal("no change built")
	}
}

// BenchmarkChangedNoMatch: a bus event kind no subscribed topic listens to.
func BenchmarkChangedNoMatch(b *testing.B) {
	route := newBody(`{"a":1}`)
	area := New(Options{Topics: []*Topic{{Name: "x", Get: route.get, Triggers: []string{"gateway_*"}}}})
	defer area.Close()
	area.subscribe("tab", []want{{topic: area.topics["x"], params: pyjson.NewObject()}}, nil)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		area.Changed("signal_like_kind")
	}
}
