package licensing

import (
	"time"
	"unicode/utf8"
)

// isoTime is what CPython 3.12's datetime.fromisoformat (C implementation) reads.
type isoTime struct {
	year, month, day, hour, minute, second, micro int
	aware                                         bool
	offsetSeconds, offsetMicro                    int
}

// isoText gives C-string access: reading past the end yields NUL, as in CPython.
type isoText string

func (s isoText) at(index int) byte {
	if index < 0 || index >= len(s) {
		return 0
	}
	return s[index]
}

func isDigitByte(c byte) bool { return c >= '0' && c <= '9' }

// digits is parse_digits: exactly count ASCII digits starting at pos.
func (s isoText) digits(pos, count int) (value, next int, ok bool) {
	for index := 0; index < count; index++ {
		c := s.at(pos + index)
		if !isDigitByte(c) {
			return 0, 0, false
		}
		value = value*10 + int(c-'0')
	}
	return value, pos + count, true
}

// separator is _find_isoformat_datetime_separator.
func (s isoText) separator() int {
	n := len(s)
	if n == 7 {
		return 7
	}
	if s.at(4) == '-' {
		if s.at(5) != 'W' {
			return 10
		}
		if n < 8 {
			return -1
		}
		if n > 8 && s.at(8) == '-' {
			if n == 9 {
				return -1
			}
			if n > 10 && isDigitByte(s.at(10)) {
				return 8
			}
			return 10
		}
		return 8
	}
	if s.at(4) != 'W' {
		return 8
	}
	index := 7
	for index < n && isDigitByte(s.at(index)) {
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

// date is parse_isoformat_date over the first length bytes.
func (s isoText) date(length int) (year, month, day int, ok bool) {
	year, p, ok := s.digits(0, 4)
	if !ok {
		return 0, 0, 0, false
	}
	separated := s.at(p) == '-'
	if separated {
		p++
	}
	if s.at(p) == 'W' {
		week, next, ok := s.digits(p+1, 2)
		if !ok {
			return 0, 0, 0, false
		}
		p, weekday := next, 1
		if p < length {
			if separated {
				if s.at(p) != '-' {
					return 0, 0, 0, false
				}
				p++
			}
			if weekday, _, ok = s.digits(p, 1); !ok {
				return 0, 0, 0, false
			}
		}
		return isoWeekDate(year, week, weekday)
	}
	month, p, ok = s.digits(p, 2)
	if !ok {
		return 0, 0, 0, false
	}
	if separated {
		if s.at(p) != '-' {
			return 0, 0, 0, false
		}
		p++
	}
	day, _, ok = s.digits(p, 2)
	return year, month, day, ok
}

// isoWeekDate is iso_to_ymd: ISO year, week and weekday to a calendar date.
func isoWeekDate(year, week, weekday int) (int, int, int, bool) {
	if year < 1 || year > 9999 {
		return 0, 0, 0, false
	}
	january1 := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	if week <= 0 || week >= 53 {
		first := (int(january1.Weekday()) + 6) % 7 // Monday = 0
		leap := year%4 == 0 && (year%100 != 0 || year%400 == 0)
		if week != 53 || !(first == 3 || (first == 2 && leap)) {
			return 0, 0, 0, false
		}
	}
	if weekday <= 0 || weekday >= 8 {
		return 0, 0, 0, false
	}
	january4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)
	monday := january4.AddDate(0, 0, -((int(january4.Weekday()) + 6) % 7))
	result := monday.AddDate(0, 0, (week-1)*7+weekday-1)
	return result.Year(), int(result.Month()), result.Day(), true
}

// parseISOTime parses text like datetime.fromisoformat; ok is false for its ValueError.
func parseISOTime(text string) (isoTime, bool) {
	s := isoText(text)
	if len(s) < 7 {
		return isoTime{}, false
	}
	at := s.separator()
	if at < 0 {
		return isoTime{}, false
	}
	var t isoTime
	var ok bool
	if t.year, t.month, t.day, ok = s.date(at); !ok {
		return isoTime{}, false
	}
	if len(s) > at {
		_, width := utf8.DecodeRuneInString(text[at:])
		if !s[at+width:].clock(&t) {
			return isoTime{}, false
		}
	}
	return t, t.valid()
}

func daysIn(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// valid applies the datetime() and timezone() range checks.
func (t isoTime) valid() bool {
	if t.year < 1 || t.year > 9999 || t.month < 1 || t.month > 12 || t.day < 1 || t.day > daysIn(t.year, t.month) {
		return false
	}
	if t.hour > 23 || t.minute > 59 || t.second > 59 {
		return false
	}
	offset := int64(t.offsetSeconds)*1_000_000 + int64(t.offsetMicro)
	return offset > -86_400_000_000 && offset < 86_400_000_000
}
