package push

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"studio-engine/internal/events"
	"studio-engine/internal/pyjson"
)

// sse reads a stream in the background so the test can wait for what arrived.
type sse struct {
	mu     sync.Mutex
	text   strings.Builder
	cancel context.CancelFunc
	done   chan struct{}
}

func openStream(t *testing.T, url string) *sse {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Fatalf("GET %s: %d", url, response.StatusCode)
	}
	s := &sse{cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(s.done)
		defer response.Body.Close()
		buffer := make([]byte, 4096)
		for {
			n, err := response.Body.Read(buffer)
			s.mu.Lock()
			s.text.Write(buffer[:n])
			s.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() { s.close() })
	return s
}

func (s *sse) close() {
	s.cancel()
	<-s.done
}

func (s *sse) wait(t *testing.T, what string, ok func(string) bool) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		s.mu.Lock()
		text := s.text.String()
		s.mu.Unlock()
		if ok(text) {
			return text
		}
		if time.Now().After(deadline) {
			t.Fatalf("waited for %s; stream so far:\n%s", what, text)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

var pushBlock = regexp.MustCompile(`event: push\ndata: [^\n]*\n\n`)

func server(t *testing.T, area *Area, bus *events.Bus) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	for _, route := range append(area.Routes(), events.Area{Bus: bus, Heartbeat: time.Hour}.Routes()...) {
		mux.Handle(route.Method+" "+route.Pattern, route.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func publish(t *testing.T, bus *events.Bus, kind string, payload *pyjson.Object) int64 {
	t.Helper()
	event, err := bus.Publish(kind, payload)
	if err != nil {
		t.Fatal(err)
	}
	return event.ID
}

func TestBusBytesIdenticalToEvents(t *testing.T) {
	area, bus := testArea(t, WallClock)
	srv := server(t, area, bus)
	for index := 0; index < 5; index++ {
		publish(t, bus, "signal", pyjson.NewObject().Set("ied_id", "ied1").Set("ref", "LD0/X.Y").Set("value", 1.5))
	}
	plain := openStream(t, srv.URL+"/api/v1/events?after=2")
	pushed := openStream(t, srv.URL+"/api/v1/push?client=tab-1&after=2")
	var last int64
	for index := 0; index < 200; index++ { // a burst, then single events
		last = publish(t, bus, "runtime", pyjson.NewObject().Set("state", "running").Set("text", "é \""))
		if index > 190 {
			time.Sleep(20 * time.Millisecond)
		}
	}
	marker := "id: " + itoa(last) + "\n"
	want := plain.wait(t, "the last event on /events", func(s string) bool { return strings.Contains(s, marker) })
	got := pushed.wait(t, "the last event on /push", func(s string) bool { return strings.Contains(s, marker) })
	for _, block := range pushBlock.FindAllString(got, -1) {
		if strings.Contains(block, "\nid:") || strings.HasPrefix(block, "id:") {
			t.Fatalf("a push frame must not carry an id: %q", block)
		}
	}
	if stripped := pushBlock.ReplaceAllString(got, ""); stripped != want {
		t.Fatalf("bus bytes differ from /api/v1/events\n/events: %q\n/push:   %q", want[:200], stripped[:200])
	}
	if !strings.Contains(got, `event: push`+"\n"+`data: {"t":"hello","epoch":"boot-1","client":"tab-1"}`) {
		t.Fatalf("hello frame missing: %q", got[:120])
	}
}

func itoa(n int64) string { b, _ := pyjson.Dumps(n); return string(b) }

func TestPushTopicsOverHTTP(t *testing.T) {
	route := newBody(`{"a":1}`)
	area, bus := testArea(t, WallClock, &Topic{Name: "health", Path: "/health", Get: route.get, Triggers: []string{"live_status"}})
	srv := server(t, area, bus)
	capabilities := call(t, http.MethodGet, srv.URL+"/api/v1/capabilities", "")
	if capabilities != `{"epoch":"boot-1","paths":{"/health":{"params":[],"t":"health"}},"push":1,"topics":["health"]}`+"\n" {
		t.Fatalf("capabilities: %q", capabilities)
	}
	answer := call(t, http.MethodPut, srv.URL+"/api/v1/push/topics",
		`{"client":"tab-1","topics":[{"t":"health"},{"t":"nope","params":{}}]}`)
	if answer != `{"epoch":"boot-1","topics":["health"],"unknown":["nope"]}`+"\n" {
		t.Fatalf("PUT answer: %q", answer)
	}
	stream := openStream(t, srv.URL+"/api/v1/push?client=tab-1")
	stream.wait(t, "the full frame", func(s string) bool {
		return strings.Contains(s, "event: push\ndata: {\"t\":\"health\",\"rev\":1,\"full\":{\"a\":1}}\n\n")
	})
	route.set(`{"a":2}`)
	publish(t, bus, "live_status", nil)
	stream.wait(t, "the changed frame and the event", func(s string) bool {
		return strings.Contains(s, `{"t":"health","rev":2,"full":{"a":2}}`) && strings.Contains(s, `"kind": "live_status"`)
	})
	if bad := call(t, http.MethodPut, srv.URL+"/api/v1/push/topics", `{"client":"no spaces"}`); !strings.Contains(bad, "error") {
		t.Fatalf("a bad client id is refused: %q", bad)
	}
}

func TestHeartbeatAndDisconnectCleanup(t *testing.T) {
	bus := events.NewBus(events.Capacity)
	route := newBody(`{"a":1}`)
	area := New(Options{Bus: bus, Heartbeat: 60 * time.Millisecond, Grace: 50 * time.Millisecond, Epoch: "e",
		Topics: []*Topic{{Name: "x", Get: route.get, Every: 10 * time.Millisecond}}})
	defer area.Close()
	srv := server(t, area, bus)
	call(t, http.MethodPut, srv.URL+"/api/v1/push/topics", `{"client":"t1","topics":[{"t":"x"}]}`)
	stream := openStream(t, srv.URL+"/api/v1/push?client=t1")
	stream.wait(t, "two heartbeats", func(s string) bool { return strings.Count(s, ": heartbeat\n\n") >= 2 })
	stream.close()
	deadline := time.Now().Add(2 * time.Second)
	for {
		area.mu.Lock()
		left := len(area.byID) + len(area.instances)
		area.mu.Unlock()
		if left == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("a closed tab must release its topics after the grace period (%d left)", left)
		}
		time.Sleep(10 * time.Millisecond)
	}
	reads := route.reads.Load()
	time.Sleep(100 * time.Millisecond)
	if route.reads.Load() != reads {
		t.Fatal("no periodic re-read may run without a subscriber")
	}
}

func TestNewStreamOfSameTabReplacesOld(t *testing.T) {
	area, bus := testArea(t, WallClock)
	srv := server(t, area, bus)
	first := openStream(t, srv.URL+"/api/v1/push?client=t1")
	first.wait(t, "hello", func(s string) bool { return strings.Contains(s, "hello") })
	second := openStream(t, srv.URL+"/api/v1/push?client=t1")
	second.wait(t, "hello", func(s string) bool { return strings.Contains(s, "hello") })
	select {
	case <-first.done:
	case <-time.After(2 * time.Second):
		t.Fatal("the older stream of a tab must end")
	}
}

func call(t *testing.T, method, url, body string) string {
	t.Helper()
	request, _ := http.NewRequest(method, url, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	text, _ := io.ReadAll(response.Body)
	return string(text)
}
