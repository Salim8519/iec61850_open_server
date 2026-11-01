package memtrim

import (
	"testing"
	"time"
)

func TestSoonCoalescesJobsIntoOneCollection(t *testing.T) {
	for i := 0; i < 5; i++ {
		Soon()
	}
	mu.Lock()
	armed := timer != nil
	mu.Unlock()
	if !armed {
		t.Fatal("no collection scheduled")
	}
	deadline := time.Now().Add(quiet + 3*time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		done := timer == nil
		mu.Unlock()
		if done {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("the scheduled collection did not run")
}

func TestHoldDelaysTheCollectionUntilTheJobEnds(t *testing.T) {
	release := Hold()
	Soon()
	mu.Lock()
	busy := held
	mu.Unlock()
	if busy != 1 {
		t.Fatalf("held = %d, want 1", busy)
	}
	release()
	release() // a second release changes nothing
	mu.Lock()
	busy, armed := held, timer != nil
	mu.Unlock()
	if busy != 0 || !armed {
		t.Fatalf("after release held=%d armed=%v, want 0 and a collection scheduled", busy, armed)
	}
}
