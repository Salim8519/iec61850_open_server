package hostkey

import (
	"testing"
	"time"
)

func TestARequesterOverTheLimitIsRefusedUntilTheWindowPasses(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	limiter := &Limiter{PerHost: 3, Total: 100, Window: time.Minute, Now: func() time.Time { return now }}
	for index := 0; index < 3; index++ {
		if !limiter.Allow("172.20.80.5") {
			t.Fatalf("request %d of 3 was refused", index+1)
		}
	}
	if limiter.Allow("172.20.80.5") {
		t.Fatal("the fourth request in a minute was answered")
	}
	if !limiter.Allow("172.20.80.6") {
		t.Fatal("another VM was punished for the first one")
	}
	now = now.Add(61 * time.Second)
	if !limiter.Allow("172.20.80.5") {
		t.Fatal("the limit never lifts")
	}
}

func TestTheTotalLimitProtectsTheHostFromManyAddresses(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	limiter := &Limiter{PerHost: 10, Total: 5, Window: time.Minute, Now: func() time.Time { return now }}
	allowed := 0
	for index := 0; index < 20; index++ {
		if limiter.Allow("10.0.0." + string(rune('a'+index))) {
			allowed++
		}
	}
	if allowed != 5 {
		t.Fatalf("%d requests were answered, the limit is 5", allowed)
	}
}

func TestTheDefaultLimitIsGenerousForPeopleAndTightForPrograms(t *testing.T) {
	limiter := DefaultLimiter()
	if limiter.PerHost < 5 || limiter.PerHost > 30 || limiter.Total < limiter.PerHost || limiter.Window != time.Minute {
		t.Fatalf("limits %+v", limiter)
	}
}
