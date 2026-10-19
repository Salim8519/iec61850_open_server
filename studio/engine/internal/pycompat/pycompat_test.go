package pycompat

import (
	"testing"
	"time"
)

// Known CPython answers, so the rules are checked even without a Python installation.
func TestKnownAnswers(t *testing.T) {
	ints := map[string]any{"+5": int64(5), " 1_000 ": int64(1000), "٣٣": int64(33),
		"-0_7": int64(-7), " 7　": int64(7)}
	for text, want := range ints {
		if got, err := ParseInt(text); err != nil || got != want {
			t.Errorf("int(%q) = %v, %v; want %v", text, got, err, want)
		}
	}
	for text, want := range map[string]string{
		"7.0":    "invalid literal for int() with base 10: '7.0'",
		"\x1c7":  "invalid literal for int() with base 10: '\\x1c7'",
		"1__0":   "invalid literal for int() with base 10: '1__0'",
		"it's":   `invalid literal for int() with base 10: "it's"`,
		"":       "invalid literal for int() with base 10: ''",
		"é": "invalid literal for int() with base 10: 'é'",
	} {
		if _, err := ParseInt(text); err == nil || err.Error() != want {
			t.Errorf("int(%q) error %v; want %s", text, err, want)
		}
	}
	if got, _ := Round(2.5); got != int64(2) {
		t.Errorf("round(2.5) = %v", got)
	}
	if got, _ := RoundDigits(2.675, 2); got != 2.67 {
		t.Errorf("round(2.675, 2) = %v", got)
	}
	if got, _ := RoundDigits(1250, -2); got != 1200 {
		t.Errorf("round(1250.0, -2) = %v", got)
	}
	if got := Casefold("Straße ΣΑΣ"); got != "strasse σασ" {
		t.Errorf("casefold = %q", got)
	}
	if got := Strip("\x1c  a b\x1f\n"); got != "a b" {
		t.Errorf("strip = %q", got)
	}
}

func TestDatesAndBytes(t *testing.T) {
	d, err := FromIsoformat("2026-10-20T08:05:03.250+00:00")
	if err != nil || !d.Aware || d.Microsecond != 250000 || d.Offset != 0 {
		t.Fatalf("fromisoformat = %+v, %v", d, err)
	}
	if got, _ := Isoformat(d, "auto"); got != "2026-10-20T08:05:03.250000+00:00" {
		t.Errorf("isoformat = %s", got)
	}
	now := FromTime(time.Date(2026, 10, 20, 8, 5, 3, 999999999, time.UTC), true)
	if got, _ := Isoformat(now, "seconds"); got != "2026-10-20T08:05:03+00:00" {
		t.Errorf("isoformat seconds = %s", got)
	}
	if _, err := FromIsoformat("2026-13-01"); err == nil || err.Error() != "month must be in 1..12" {
		t.Errorf("month error = %v", err)
	}
	if text, err := DecodeUTF8Sig([]byte("\xef\xbb\xbfa,b")); err != nil || text != "a,b" {
		t.Errorf("utf-8-sig = %q, %v", text, err)
	}
	want := "'utf-8' codec can't decode byte 0xff in position 1: invalid start byte"
	if _, err := DecodeUTF8Sig([]byte("\xef\xbb\xbfa\xff")); err == nil || err.Error() != want {
		t.Errorf("utf-8-sig error = %v", err)
	}
	if got := TimeTime(time.Unix(1760947200, 500000000)); got != 1760947200.5 {
		t.Errorf("time.time() = %v", got)
	}
}
