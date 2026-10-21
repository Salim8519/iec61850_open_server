package push

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"studio-engine/internal/events"
	"studio-engine/internal/pyjson"
)

// body is a route stand-in whose answer the test sets; it counts the reads.
type body struct {
	mu     sync.Mutex
	status int
	text   string
	reads  atomic.Int64
}

func newBody(text string) *body { return &body{status: 200, text: text} }

func (b *body) set(text string) {
	b.mu.Lock()
	b.text = text
	b.mu.Unlock()
}

func (b *body) get(query string) (int, []byte) {
	b.reads.Add(1)
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.status, []byte(b.text + "\n") // Flask bodies end with a newline
}

// testArea is an Area on a fake clock with one bus.
func testArea(t *testing.T, clock Clock, topics ...*Topic) (*Area, *events.Bus) {
	t.Helper()
	bus := events.NewBus(events.Capacity)
	area := New(Options{Bus: bus, Topics: topics, Clock: clock, Epoch: "boot-1", Heartbeat: time.Hour})
	t.Cleanup(area.Close)
	return area, bus
}

// tab is a client attached without HTTP: the test drains its outbox directly.
type tab struct {
	t    *testing.T
	area *Area
	c    *client
	gen  int64
}

func attachTab(t *testing.T, area *Area, id string) *tab {
	t.Helper()
	c, gen, _, err := area.attach(id)
	if err != nil {
		t.Fatal(err)
	}
	return &tab{t: t, area: area, c: c, gen: gen}
}

func (tb *tab) subscribe(names ...string) {
	tb.t.Helper()
	var wanted []want
	for _, name := range names {
		wanted = append(wanted, want{topic: tb.area.topics[name]})
	}
	if err := tb.area.subscribe(tb.c.id, wanted, nil); err != nil {
		tb.t.Fatal(err)
	}
}

func (tb *tab) subscribeWith(name string, params *pyjson.Object) {
	tb.t.Helper()
	if err := tb.area.subscribe(tb.c.id, []want{{topic: tb.area.topics[name], params: params}}, nil); err != nil {
		tb.t.Fatal(err)
	}
}

// frames waits up to 2 s for n frames (decoded) and fails on more or fewer.
func (tb *tab) frames(n int) []map[string]any {
	tb.t.Helper()
	var got []map[string]any
	deadline := time.Now().Add(2 * time.Second)
	for {
		raw, ok := tb.c.drain(tb.gen, nil)
		if !ok {
			tb.t.Fatal("stream superseded")
		}
		for _, frame := range raw {
			var decoded map[string]any
			if err := json.Unmarshal(frame, &decoded); err != nil {
				tb.t.Fatalf("frame %s: %v", frame, err)
			}
			decoded["_raw"] = string(frame)
			got = append(got, decoded)
		}
		if len(got) >= n && (n > 0 || time.Now().After(deadline.Add(-1900*time.Millisecond))) {
			break
		}
		if time.Now().After(deadline) {
			tb.t.Fatalf("waited for %d frames, got %d: %v", n, len(got), got)
		}
		time.Sleep(2 * time.Millisecond)
	}
	if len(got) != n {
		tb.t.Fatalf("want %d frames, got %d: %v", n, len(got), got)
	}
	return got
}

// settle lets queued builds finish (the build goroutine is real even on a fake clock).
func settle() { time.Sleep(30 * time.Millisecond) }

func num(value any) int64 {
	number, _ := value.(float64)
	return int64(number)
}
