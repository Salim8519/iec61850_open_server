package pyv

import (
	"time"
)

// ISOSeconds is datetime.now(timezone.utc).isoformat(timespec="seconds") for t,
// e.g. "2026-10-20T08:15:02+00:00".
func ISOSeconds(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05") + "+00:00"
}

var isoLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05.999999Z07:00",
	"2006-01-02T15:04Z07:00",
	"2006-01-02T15Z07:00",
}

// ParseAware parses an ISO timestamp with a UTC offset, as datetime.fromisoformat does for
// the values Studio writes. It reports false for text without an offset (Python would give
// a naive datetime, and subtracting it from an aware one raises TypeError) or bad text.
func ParseAware(text string) (time.Time, bool) {
	for _, layout := range isoLayouts {
		if parsed, err := time.Parse(layout, text); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
