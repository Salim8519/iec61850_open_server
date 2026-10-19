package health

import (
	"time"
	"unicode/utf8"
)

// ageMicros is _age in whole microseconds: now minus datetime.fromisoformat(observed).
// It reports false where Python returns None: a value that is not a string, text that
// fromisoformat rejects, or a timestamp without UTC offset (naive minus aware raises
// TypeError). now counts with microsecond precision, as Python's datetime does.
func ageMicros(observed any, now time.Time) (int64, bool) {
	text, isText := observed.(string)
	if !isText {
		return 0, false
	}
	at, aware, ok := fromISOFormat(text)
	if !ok || !aware {
		return 0, false
	}
	seconds := now.Unix() - at.Unix()
	return seconds*1_000_000 + int64(now.Nanosecond()/1000) - int64(at.Nanosecond()/1000), true
}

// isoText reads like the NUL-terminated C string CPython parses: past the end is 0.
type isoText string

func (s isoText) at(index int) byte {
	if index >= 0 && index < len(s) {
		return s[index]
	}
	return 0
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// digits is parse_digits: count ASCII digits from index as a number, and the next index.
func (s isoText) digits(index, count int) (int, int, bool) {
	value := 0
	for i := 0; i < count; i++ {
		c := s.at(index + i)
		if !isDigit(c) {
			return 0, 0, false
		}
		value = value*10 + int(c-'0')
	}
	return value, index + count, true
}

// fromISOFormat mirrors CPython 3.12's C datetime.fromisoformat on the UTF-8 bytes of
// text: the instant (UTC), whether the text had an offset (aware), and whether it parsed.
// Python replaces a lone surrogate separator with "T"; Go strings cannot hold one.
func fromISOFormat(text string) (time.Time, bool, bool) {
	if utf8.RuneCountInString(text) < 7 {
		return time.Time{}, false, false
	}
	s := isoText(text)
	separator := separatorAt(s)
	if separator < 0 {
		return time.Time{}, false, false
	}
	year, month, day, ok := parseDate(s, separator)
	if !ok {
		return time.Time{}, false, false
	}
	var clock [4]int
	var offset time.Duration
	aware := false
	if len(s) > separator {
		start := separator + 1
		for start < len(s) && s[start]&0xC0 == 0x80 {
			start++ // a non-ASCII separator spans several UTF-8 bytes
		}
		if clock, offset, aware, ok = parseTime(s, start); !ok {
			return time.Time{}, false, false
		}
	}
	if !validDate(year, month, day) || clock[0] > 23 || clock[1] > 59 || clock[2] > 59 {
		return time.Time{}, false, false
	}
	if offset <= -24*time.Hour || offset >= 24*time.Hour {
		return time.Time{}, false, false
	}
	at := time.Date(year, time.Month(month), day, clock[0], clock[1], clock[2], clock[3]*1000, time.UTC)
	return at.Add(-offset), aware, true
}

// separatorAt is _find_isoformat_datetime_separator: where the date part ends.
func separatorAt(s isoText) int {
	if len(s) == 7 {
		return 7
	}
	if s.at(4) == '-' {
		if s.at(5) != 'W' {
			return 10 // YYYY-MM-DD
		}
		switch {
		case len(s) < 8:
			return -1
		case len(s) == 8:
			return 8 // YYYY-Www
		case s.at(8) != '-':
			return 8
		case len(s) == 9:
			return -1
		case len(s) > 10 && isDigit(s.at(10)):
			return 8
		}
		return 10 // YYYY-Www-D
	}
	if s.at(4) != 'W' {
		return 8 // YYYYMMDD
	}
	index := 7
	for index < len(s) && isDigit(s.at(index)) {
		index++
	}
	if index < 9 {
		return index
	}
	if index%2 == 0 {
		return 7
	}
	return 8
}

// parseDate is parse_isoformat_date over the first length bytes (calendar or ISO week).
func parseDate(s isoText, length int) (int, int, int, bool) {
	year, p, ok := s.digits(0, 4)
	if !ok {
		return 0, 0, 0, false
	}
	dashes := s.at(p) == '-'
	if dashes {
		p++
	}
	if s.at(p) == 'W' {
		week, next, ok := s.digits(p+1, 2)
		if !ok {
			return 0, 0, 0, false
		}
		weekday := 1
		if next < length {
			if dashes {
				if s.at(next) != '-' {
					return 0, 0, 0, false
				}
				next++
			}
			if weekday, _, ok = s.digits(next, 1); !ok {
				return 0, 0, 0, false
			}
		}
		return isoWeekDate(year, week, weekday)
	}
	month, p, ok := s.digits(p, 2)
	if !ok {
		return 0, 0, 0, false
	}
	if dashes {
		if s.at(p) != '-' {
			return 0, 0, 0, false
		}
		p++
	}
	day, _, ok := s.digits(p, 2)
	return year, month, day, ok
}

// isoWeekDate is iso_to_ymd: the calendar date of an ISO year, week and weekday.
func isoWeekDate(year, week, weekday int) (int, int, int, bool) {
	if year < 1 || year > 9999 {
		return 0, 0, 0, false
	}
	january := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
	first := (int(january.Weekday()) + 6) % 7 // Monday is 0
	if week <= 0 || week >= 53 {
		leap := year%4 == 0 && (year%100 != 0 || year%400 == 0)
		if week != 53 || !(first == 3 || (first == 2 && leap)) {
			return 0, 0, 0, false
		}
	}
	if weekday <= 0 || weekday >= 8 {
		return 0, 0, 0, false
	}
	monday := january.AddDate(0, 0, -first)
	if first > 3 {
		monday = monday.AddDate(0, 0, 7)
	}
	date := monday.AddDate(0, 0, (week-1)*7+weekday-1)
	return date.Year(), int(date.Month()), date.Day(), true
}

// validDate is the year/month/day range check of the datetime constructor.
func validDate(year, month, day int) bool {
	if year < 1 || year > 9999 || month < 1 || month > 12 || day < 1 {
		return false
	}
	last := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	return day <= last
}
