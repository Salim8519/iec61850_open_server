package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recorder collects hub changes.
type recorder struct {
	mu   sync.Mutex
	seen map[Change]int
}

func (r *recorder) note(change Change) {
	r.mu.Lock()
	r.seen[change]++
	r.mu.Unlock()
}

func (r *recorder) reset() {
	r.mu.Lock()
	r.seen = map[Change]int{}
	r.mu.Unlock()
}

// expect waits until every change was reported.
func (r *recorder) expect(t *testing.T, step string, changes ...Change) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		r.mu.Lock()
		var missing []Change
		for _, change := range changes {
			if r.seen[change] == 0 {
				missing = append(missing, change)
			}
		}
		r.mu.Unlock()
		if len(missing) == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s: not reported: %+v", step, missing)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestHubCoalescesAndResyncsWithoutBlocking(t *testing.T) {
	var h hub
	release := make(chan struct{})
	got := make(chan Change, 16)
	first := true
	cancel := h.subscribe(func(change Change) {
		if first {
			first = false
			<-release
		}
		got <- change
	})
	defer cancel()
	h.notify(Change{"a", "blocker"})
	time.Sleep(20 * time.Millisecond) // the subscriber now waits inside notify
	for i := 0; i < 100; i++ {
		h.notify(Change{"a", "same"})
	}
	close(release)
	if change := <-got; change != (Change{"a", "blocker"}) {
		t.Fatalf("first %+v", change)
	}
	if change := <-got; change != (Change{"a", "same"}) {
		t.Fatalf("coalesced %+v", change)
	}
	select {
	case change := <-got:
		t.Fatalf("repeated change delivered twice: %+v", change)
	case <-time.After(50 * time.Millisecond):
	}
	// Overflow: a stuck subscriber never blocks notify; it gets one resync instead.
	stuck := make(chan struct{})
	var resyncs atomic.Int64
	var calls atomic.Int64
	cancelStuck := h.subscribe(func(change Change) {
		if calls.Add(1) == 1 {
			<-stuck
		}
		if change == (Change{}) {
			resyncs.Add(1)
		}
	})
	defer cancelStuck()
	h.notify(Change{"b", "first"})
	time.Sleep(20 * time.Millisecond)
	started := time.Now()
	for i := 0; i < hubQueue+100; i++ {
		h.notify(Change{"b", fmt.Sprint(i)})
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("notify blocked for %v", elapsed)
	}
	close(stuck)
	deadline := time.Now().Add(2 * time.Second)
	for resyncs.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if resyncs.Load() != 1 || calls.Load() != 2 {
		t.Fatalf("resyncs %d calls %d, want 1 resync after the first change", resyncs.Load(), calls.Load())
	}
}

func TestHubCancelInsideNotify(t *testing.T) {
	var h hub
	var calls atomic.Int64
	var cancel func()
	ready := make(chan struct{})
	cancel = h.subscribe(func(Change) {
		<-ready
		calls.Add(1)
		cancel()
	})
	close(ready)
	h.notify(Change{"a", "1"})
	time.Sleep(20 * time.Millisecond)
	h.notify(Change{"a", "2"})
	time.Sleep(20 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatalf("calls after cancel: %d", calls.Load())
	}
}

// BenchmarkHubNotify measures one notify (a value write's cost) with warm queues; it must
// not allocate (S6).
func BenchmarkHubNotify(b *testing.B) {
	changes := make([]Change, 256)
	for i := range changes {
		changes[i] = Change{IEDID: "ied", Ref: fmt.Sprintf("LD0/GGIO1.AnIn%d.mag.f", i)}
	}
	for _, count := range []int{0, 1, 4} {
		b.Run(fmt.Sprintf("subscribers=%d", count), func(b *testing.B) {
			var h hub
			var delivered atomic.Int64
			for i := 0; i < count; i++ {
				cancel := h.subscribe(func(Change) { delivered.Add(1) })
				defer cancel()
			}
			for round := 0; round < 20; round++ { // grow queues and coalescing maps
				for _, change := range changes {
					h.notify(change)
				}
				time.Sleep(time.Millisecond)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				h.notify(changes[i&255])
			}
		})
	}
}
