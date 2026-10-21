package updates

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestRealHTTPWithETag runs the check over net/http against a local httptest server: the
// stored ETag goes back as If-None-Match and a 304 keeps the last answer.
func TestRealHTTPWithETag(t *testing.T) {
	var mu sync.Mutex
	var seen []http.Header
	signed := *load(t).Scenarios[0].Steps[1].Answers[0].Body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Header.Clone())
		mu.Unlock()
		if r.URL.Path != "/api/v1/release" || r.Method != "GET" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("If-None-Match") == `"e1"` {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"e1"`)
		w.Write([]byte(signed))
	}))
	defer server.Close()
	clock := 1_000_000.0
	service := New(Options{Directory: t.TempDir(), ServerURL: func() (string, error) { return server.URL + "/", nil },
		Current: "0.1.13", Keys: testKeys(t), Opener: server.Client(), Clock: func() float64 { return clock }})
	first, err := service.Check(false)
	if err != nil || get(first, "available") != true || get(first, "error") != nil {
		t.Fatalf("first check %v %v", first, err)
	}
	clock += Day
	second, err := service.Check(false)
	if err != nil || get(second, "available") != true || get(second, "checked_at") != int64(clock) {
		t.Fatalf("second check %v %v", second, err)
	}
	if len(seen) != 2 || seen[1].Get("If-None-Match") != `"e1"` || seen[0].Get("User-Agent") != "ScadaBench" {
		t.Fatalf("requests %v", seen)
	}
	for _, name := range []string{"Cookie", "Authorization", "Content-Type"} {
		if seen[0].Get(name) != "" {
			t.Errorf("the check must send nothing about the PC (%s)", name)
		}
	}
}

func TestBackgroundScheduleWithoutWaiting(t *testing.T) {
	clock := 1_000_000.0
	calls := 0
	opener := &cannedOpener{}
	service := New(Options{Directory: t.TempDir(), ServerURL: func() (string, error) { return "ftp://none", nil },
		Current: "0.1.13", Opener: opener, Clock: func() float64 { return clock }})
	var slept []float64
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	schedule := Schedule{
		Uniform: func(low, high float64) float64 { return high },
		Sleep: func(_ context.Context, seconds float64) bool {
			slept = append(slept, seconds)
			clock += seconds
			calls++
			return calls < 30
		},
	}
	service.Run(ctx, schedule)
	if slept[0] != StartDelayMax || slept[1] != 3600+600 {
		t.Fatalf("delays %v", slept[:2])
	}
	status, _ := service.Status()
	if get(status, "error") != "no_server" {
		t.Fatalf("background check did not run: %v", status)
	}
	// 29 hourly wake-ups over ~32 h: a check at start and one a day later.
	checked := get(status, "checked_at").(int64)
	if float64(checked) > clock || clock-float64(checked) >= Day {
		t.Fatalf("checked_at %d, clock %v", checked, clock)
	}
	t.Setenv("SCADABENCH_UPDATE_CHECK", "off")
	if !Disabled() {
		t.Fatal("SCADABENCH_UPDATE_CHECK=off must disable the loop")
	}
	service.StartBackground(ctx) // returns at once; nothing runs
}
