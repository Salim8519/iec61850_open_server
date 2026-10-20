package app

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"studio-engine/internal/testutil"
)

// stressLane is one kind of work repeated until the test ends; progress counts finished
// steps so the watchdog can tell a stall from slow work.
type stressLane struct {
	name     string
	step     func() int // HTTP status of the step (0 when not an HTTP step)
	progress atomic.Int64
	failures atomic.Int64
}

// TestComposedLockOrderStress (STUDIO_STRESS=1, real workers): Run/Stop, canvas operates,
// gateway saves, project edits, preferences, readers and a live SSE client all at once on
// the composed engine for STUDIO_STRESS_SECONDS (default 60). A lane without progress for
// 10 s dumps every goroutine and fails; a 500 answer fails.
func TestComposedLockOrderStress(t *testing.T) {
	if os.Getenv("STUDIO_STRESS") != "1" {
		t.Skip("set STUDIO_STRESS=1 to run the composed stress test")
	}
	seconds := 60
	if value, err := strconv.Atoi(os.Getenv("STUDIO_STRESS_SECONDS")); err == nil && value > 0 {
		seconds = value
	}
	engine := composeAt(t, testutil.TempDir(t), muxNative(t))
	defer engine.Close()
	id := clickProject(t, engine)
	status := func(method, path string, body any) int { code, _ := call(t, engine, method, path, body); return code }
	operateValue := atomic.Bool{}
	lanes := []*stressLane{
		{name: "lifecycle", step: func() int {
			code := status("POST", "/api/v1/runtime/start", nil)
			time.Sleep(300 * time.Millisecond)
			if code == 200 {
				code = status("POST", "/api/v1/runtime/stop", nil)
			}
			return code
		}},
		{name: "operate", step: func() int {
			closeIt := !operateValue.Load()
			operateValue.Store(closeIt)
			return status("POST", "/api/v1/equipment/CB1/operate", obj("value", closeIt))
		}},
		{name: "gateway", step: func() int {
			return status("PUT", "/api/v1/tools/gateway/config", obj("protocol", "modbus", "listen", "this_pc"))
		}},
		{name: "edits", step: func() int {
			code := status("PUT", "/api/v1/ieds/"+id+"/settings", obj("goose_publish", true))
			snapshot := mustCall(t, engine, "GET", "/api/v1/project", nil)
			status("PUT", "/api/v1/project/diagram", obj("diagram", field(snapshot, "diagram"),
				"revision", field(snapshot, "revision"), "tab_id", "stress"))
			status("PUT", "/api/v1/preferences", obj("theme", "dark"))
			return code
		}},
		{name: "readers", step: func() int {
			for _, path := range []string{"/api/v1/project", "/api/v1/equipment/states", "/api/v1/mms/access",
				"/api/v1/live", "/api/v1/goose", "/api/v1/project/test-panel", "/api/v1/tools/gateway"} {
				if code := status("GET", path, nil); code >= 500 && code != 503 {
					return code
				}
			}
			return 200
		}},
	}
	sse := &stressLane{name: "sse"}
	stop := make(chan struct{})
	var group sync.WaitGroup
	for _, lane := range lanes {
		group.Add(1)
		go func(lane *stressLane) {
			defer group.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				if code := lane.step(); code == http.StatusInternalServerError {
					lane.failures.Add(1)
				}
				lane.progress.Add(1)
			}
		}(lane)
	}
	closeSSE := streamEvents(t, engine, sse, stop)
	watched := append(lanes, sse)
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	last, lastAt := make([]int64, len(watched)), make([]time.Time, len(watched))
	for i := range lastAt {
		lastAt[i] = time.Now()
	}
	for time.Now().Before(deadline) && !t.Failed() {
		time.Sleep(500 * time.Millisecond)
		for i, lane := range watched {
			if now := lane.progress.Load(); now != last[i] {
				last[i], lastAt[i] = now, time.Now()
			} else if time.Since(lastAt[i]) > 10*time.Second {
				var dump strings.Builder
				_ = pprof.Lookup("goroutine").WriteTo(&dump, 2)
				t.Errorf("lane %s made no progress for 10 s; goroutines:\n%s", lane.name, dump.String())
				break
			}
		}
	}
	close(stop)
	closeSSE()
	group.Wait()
	status("POST", "/api/v1/runtime/stop", nil)
	var summary []string
	for _, lane := range watched {
		summary = append(summary, lane.name+" "+strconv.FormatInt(lane.progress.Load(), 10))
		if lane.failures.Load() > 0 {
			t.Errorf("lane %s got %d answers with status 500", lane.name, lane.failures.Load())
		}
	}
	t.Logf("%d s: %s; %d events", seconds, strings.Join(summary, ", "), engine.Bus.Sequence())
}

// streamEvents reads the real SSE stream over loopback HTTP; every frame is progress.
func streamEvents(t *testing.T, engine *Engine, lane *stressLane, stop chan struct{}) func() {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &httptest.Server{Listener: listener, Config: &http.Server{Handler: engine.Handler}}
	server.Start()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			request, _ := http.NewRequest("GET", server.URL+"/api/v1/events", nil)
			request.Header.Set("Authorization", "Bearer token")
			response, err := http.DefaultClient.Do(request)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}
			lines := bufio.NewScanner(response.Body)
			for lines.Scan() {
				if strings.HasPrefix(lines.Text(), "data: ") {
					lane.progress.Add(1)
				}
			}
			response.Body.Close()
		}
	}()
	return func() {
		server.CloseClientConnections()
		server.Close()
		<-done
	}
}
