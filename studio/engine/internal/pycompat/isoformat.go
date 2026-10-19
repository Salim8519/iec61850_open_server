package pycompat

import (
	"fmt"
	"math"
	"strings"
	"time"

	"studio-engine/internal/apperr"
)

// DateTime is a Python datetime: wall-clock fields and, when Aware, a UTC offset
// (Python allows offsets with seconds and microseconds, strictly inside ±24 h).
type DateTime struct {
	Year, Month, Day                  int
	Hour, Minute, Second, Microsecond int
	Aware                             bool
	Offset                            time.Duration
}

// FromTime returns the datetime of t in t's location, with microseconds truncated as
// datetime.now() does; aware keeps t's UTC offset (datetime.now(timezone.utc) is aware).
func FromTime(t time.Time, aware bool) DateTime {
	_, offset := t.Zone()
	return DateTime{Year: t.Year(), Month: int(t.Month()), Day: t.Day(), Hour: t.Hour(),
		Minute: t.Minute(), Second: t.Second(), Microsecond: t.Nanosecond() / 1000,
		Aware: aware, Offset: time.Duration(offset) * time.Second}
}

// Time returns the instant (naive datetimes are taken as UTC wall clock).
func (d DateTime) Time() time.Time {
	zone := time.UTC
	if d.Aware && d.Offset != 0 {
		zone = time.FixedZone("", int(d.Offset/time.Second))
	}
	return time.Date(d.Year, time.Month(d.Month), d.Day, d.Hour, d.Minute, d.Second,
		d.Microsecond*1000, zone)
}

// Isoformat is datetime.isoformat(timespec=...) with the 'T' separator; timespec is
// "auto", "hours", "minutes", "seconds", "milliseconds" or "microseconds".
func Isoformat(d DateTime, timespec string) (string, error) {
	date := fmt.Sprintf("%04d-%02d-%02dT", d.Year, d.Month, d.Day)
	if timespec == "auto" {
		timespec = "seconds"
		if d.Microsecond != 0 {
			timespec = "microseconds"
		}
	}
	var clock string
	switch timespec {
	case "hours":
		clock = fmt.Sprintf("%02d", d.Hour)
	case "minutes":
		clock = fmt.Sprintf("%02d:%02d", d.Hour, d.Minute)
	case "seconds":
		clock = fmt.Sprintf("%02d:%02d:%02d", d.Hour, d.Minute, d.Second)
	case "milliseconds":
		clock = fmt.Sprintf("%02d:%02d:%02d.%03d", d.Hour, d.Minute, d.Second, d.Microsecond/1000)
	case "microseconds":
		clock = fmt.Sprintf("%02d:%02d:%02d.%06d", d.Hour, d.Minute, d.Second, d.Microsecond)
	default:
		return "", apperr.ValueError("Unknown timespec value")
	}
	if !d.Aware {
		return date + clock, nil
	}
	return date + clock + FormatUTCOffset(d.Offset), nil
}

// FormatUTCOffset writes an offset as datetime does: ±HH:MM, plus :SS and .ffffff only
// when they are not zero.
func FormatUTCOffset(offset time.Duration) string {
	sign := "+"
	if offset < 0 {
		sign, offset = "-", -offset
	}
	micro := int64(offset / time.Microsecond)
	seconds, micro := micro/1e6, micro%1e6
	hours, minutes, seconds := seconds/3600, seconds/60%60, seconds%60
	switch {
	case micro != 0:
		return fmt.Sprintf("%s%02d:%02d:%02d.%06d", sign, hours, minutes, seconds, micro)
	case seconds != 0:
		return fmt.Sprintf("%s%02d:%02d:%02d", sign, hours, minutes, seconds)
	}
	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

// TimeTime is time.time() for t: float seconds since the epoch, computed from integer
// nanoseconds as CPython does.
func TimeTime(t time.Time) float64 {
	nanos := t.UnixNano()
	if nanos%1e9 == 0 {
		return float64(nanos / 1e9)
	}
	return float64(nanos) / 1e9
}

// FromTimestamp is datetime.fromtimestamp(seconds, timezone.utc) on Windows: the
// fraction rounds to microseconds with halves to even; the CRT accepts whole seconds from
// -43200 to 32536850399 only (OSError outside), beyond time_t it is an OverflowError.
func FromTimestamp(seconds float64) (DateTime, error) {
	if math.IsNaN(seconds) {
		return DateTime{}, apperr.ValueError("Invalid value NaN (not a number)")
	}
	whole, fraction := math.Modf(seconds)
	micro := math.RoundToEven(fraction * 1e6)
	if micro >= 1e6 {
		micro -= 1e6
		whole++
	} else if micro < 0 {
		micro += 1e6
		whole--
	}
	if !(whole >= -(1<<63) && whole < 1<<63) {
		return DateTime{}, overflow("timestamp out of range for platform time_t")
	}
	if whole < -43200 || whole > 32536850399 {
		return DateTime{}, &Error{Class: "OSError", Message: "[Errno 22] Invalid argument"}
	}
	t := time.Unix(int64(whole), int64(micro)*1000).UTC()
	return FromTime(t, true), nil
}

// TimedeltaRepr is repr(timedelta) for a duration with microsecond resolution, e.g.
// "datetime.timedelta(days=-1, seconds=86340)".
func TimedeltaRepr(value time.Duration) string {
	micro := int64(value / time.Microsecond)
	days := floorDiv(micro, 86400*1e6)
	micro -= days * 86400 * 1e6
	seconds, micro := micro/1e6, micro%1e6
	var parts []string
	if days != 0 {
		parts = append(parts, fmt.Sprintf("days=%d", days))
	}
	if seconds != 0 {
		parts = append(parts, fmt.Sprintf("seconds=%d", seconds))
	}
	if micro != 0 {
		parts = append(parts, fmt.Sprintf("microseconds=%d", micro))
	}
	if len(parts) == 0 {
		return "datetime.timedelta(0)"
	}
	return "datetime.timedelta(" + strings.Join(parts, ", ") + ")"
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}
