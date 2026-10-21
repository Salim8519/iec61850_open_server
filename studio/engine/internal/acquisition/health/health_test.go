package health

import (
	"strings"
	"testing"
	"time"
)

func TestBuildHealthWithMissingInputs(t *testing.T) {
	now := time.Date(2026, 10, 20, 8, 0, 0, 0, time.UTC)
	got := BuildHealth(nil, true, nil, nil, now)
	if text := CSVText(got); text != "ied,role,area,item,result,message,observed_at,source,stale\n" {
		t.Fatalf("CSVText = %q", text)
	}
	if html := HTMLReport(got, "a&b", "r"); !strings.Contains(html, "Station health: a&amp;b") ||
		!strings.Contains(html, "<strong>Not checked</strong>") {
		t.Fatalf("HTMLReport = %s", html)
	}
}

func TestIsoSecondsKeepsOffset(t *testing.T) {
	zone := time.FixedZone("x", -(5*3600 + 30*60 + 15))
	if got := isoSeconds(time.Date(2026, 1, 2, 3, 4, 5, 999, zone)); got != "2026-01-02T03:04:05-05:30:15" {
		t.Fatalf("isoSeconds = %s", got)
	}
}

func TestCSVFieldQuoting(t *testing.T) {
	cases := map[string]string{"a b": "a b", "x,y": `"x,y"`, `q"q`: `"q""q"`, "n\nl": "\"n\nl\"", "r\rl": "\"r\rl\"", "": ""}
	for in, want := range cases {
		if got := csvField(in); got != want {
			t.Errorf("csvField(%q) = %q, want %q", in, got, want)
		}
	}
}
