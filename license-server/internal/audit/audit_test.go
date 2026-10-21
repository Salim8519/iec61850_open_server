package audit

import (
	"testing"
	"time"
)

func TestRecentReturnsNewestFirstAndKeepsEverything(t *testing.T) {
	log := Open(t.TempDir())
	if entries, err := log.Recent(5); err != nil || len(entries) != 0 {
		t.Fatal("empty log must read as empty")
	}
	for i := 0; i < 8; i++ {
		if err := log.Add("login", "1.2.3.4", "", i%2 == 0, time.Unix(int64(100+i), 0)); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := log.Recent(3)
	if err != nil || len(entries) != 3 || entries[0].Time != 107 || entries[2].Time != 105 {
		t.Fatalf("wrong recent entries %+v %v", entries, err)
	}
	all, _ := log.Recent(100)
	if len(all) != 8 {
		t.Fatal("every entry must be kept")
	}
}
