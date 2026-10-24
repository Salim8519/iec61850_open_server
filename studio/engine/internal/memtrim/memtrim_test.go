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
