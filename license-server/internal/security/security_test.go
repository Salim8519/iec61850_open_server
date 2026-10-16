package security

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func solved(secret string, difficulty int, now time.Time) Solution {
	c := IssueChallenge(secret, difficulty, now)
	return Solution{Nonce: c.Nonce, Timestamp: c.Timestamp, Difficulty: c.Difficulty, MAC: c.MAC, Counter: Solve(c)}
}

func TestPowAcceptsSolvedChallengeOnce(t *testing.T) {
	now := time.Unix(1_760_000_000, 0)
	v := NewVerifier("secret", 8)
	s := solved("secret", 8, now)
	if err := v.Verify(s, now.Add(time.Second)); err != nil {
		t.Fatalf("valid solution rejected: %v", err)
	}
	if err := v.Verify(s, now.Add(2*time.Second)); err == nil {
		t.Fatal("replayed solution accepted")
	}
}

func TestPowRejectsForgedExpiredEasyAndUnsolved(t *testing.T) {
	now := time.Unix(1_760_000_000, 0)
	v := NewVerifier("secret", 8)
	forged := solved("other-secret", 8, now)
	if v.Verify(forged, now) == nil {
		t.Fatal("challenge signed with another secret accepted")
	}
	if v.Verify(solved("secret", 8, now), now.Add(6*time.Minute)) == nil {
		t.Fatal("expired challenge accepted")
	}
	if v.Verify(solved("secret", 4, now), now) == nil {
		t.Fatal("challenge below minimum difficulty accepted")
	}
	unsolved := solved("secret", 8, now)
	for counter := uint64(0); ; counter++ {
		digest := Digest(unsolved.Nonce, counter)
		if LeadingZeroBits(digest[:]) < 8 {
			unsolved.Counter = counter
			break
		}
	}
	if v.Verify(unsolved, now) == nil {
		t.Fatal("unsolved counter accepted")
	}
}

func TestLeadingZeroBits(t *testing.T) {
	cases := map[string]int{"\x00\x00\x80": 16, "\x00\x01": 15, "\xff": 0, "\x00\x00": 16}
	for input, expected := range cases {
		if got := LeadingZeroBits([]byte(input)); got != expected {
			t.Errorf("%q: got %d want %d", input, got, expected)
		}
	}
}

func TestLimiterBucketRefillAndBan(t *testing.T) {
	cfg := DefaultLimiterConfig()
	l := NewLimiter(cfg)
	now := time.Unix(1_760_000_000, 0)
	for i := 0; i < 5; i++ {
		if !l.Allow("1.2.3.4", ClassSensitive, now) {
			t.Fatalf("request %d refused early", i)
		}
	}
	if l.Allow("1.2.3.4", ClassSensitive, now) {
		t.Fatal("sixth sensitive request in a minute allowed")
	}
	if !l.Allow("1.2.3.4", ClassSensitive, now.Add(13*time.Second)) {
		t.Fatal("bucket did not refill")
	}
	for i := 0; i < cfg.MaxFailures; i++ {
		l.Fail("5.6.7.8", now)
	}
	if !l.Banned("5.6.7.8", now) || l.Allow("5.6.7.8", ClassGeneral, now.Add(29*time.Minute)) {
		t.Fatal("IP not banned after repeated failures")
	}
	if !l.Allow("5.6.7.8", ClassGeneral, now.Add(31*time.Minute)) {
		t.Fatal("ban did not expire")
	}
}

func TestLimiterMemoryIsBounded(t *testing.T) {
	cfg := DefaultLimiterConfig()
	cfg.MaxEntries = 100
	l := NewLimiter(cfg)
	now := time.Unix(1_760_000_000, 0)
	for i := 0; i < 1000; i++ {
		l.Allow(fmt.Sprintf("10.0.%d.%d", i/256, i%256), ClassGeneral, now)
	}
	if l.Len() != 100 {
		t.Fatalf("limiter holds %d entries, want 100", l.Len())
	}
}

func TestClientIPTrustsCloudflareOnlyFromLoopback(t *testing.T) {
	local := httptest.NewRequest("GET", "/", nil)
	local.RemoteAddr = "127.0.0.1:5000"
	local.Header.Set("CF-Connecting-IP", "203.0.113.9")
	if ClientIP(local, true) != "203.0.113.9" {
		t.Fatal("tunnel client IP ignored")
	}
	remote := httptest.NewRequest("GET", "/", nil)
	remote.RemoteAddr = "198.51.100.7:5000"
	remote.Header.Set("CF-Connecting-IP", "203.0.113.9")
	if ClientIP(remote, true) != "198.51.100.7" {
		t.Fatal("spoofed CF header from a non-loopback peer trusted")
	}
}
