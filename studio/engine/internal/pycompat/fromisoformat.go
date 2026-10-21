package pycompat

import (
	"fmt"
	"time"
	"unicode/utf8"

	"studio-engine/internal/apperr"
)

// FromIsoformat is datetime.fromisoformat(text) of CPython 3.12 (the C implementation):
// YYYY-MM-DD, YYYYMMDD and ISO week dates, any single separator character, then
// HH[:MM[:SS[.f...]]] or the basic form, fractions with '.' or ',' (digits beyond six
// dropped), and Z or ±HH[:MM[:SS[.ffffff]]] offsets. Errors are CPython's ValueErrors.
func FromIsoformat(text string) (DateTime, error) {
	invalid := apperr.ValueError("Invalid isoformat string: " + Repr(text))
	if utf8.RuneCountInString(text) < 7 || !utf8.ValidString(text) {
		return DateTime{}, invalid
	}
	s := isoText(sanitizeSeparator(text))
	separator := s.separatorLocation()
	var d DateTime
	ok := s.parseDate(separator, &d)
	if ok && len(s) > separator {
		ok = s[separator+1:].parseTime(&d)
	}
	if !ok {
		return DateTime{}, invalid
	}
	if err := checkDateTime(d); err != nil {
		return DateTime{}, err
	}
	return d, nil
}

// sanitizeSeparator accepts any character as the date/time separator, as CPython does: a
// non-ASCII character at one of the possible separator positions (7, 8, 10, the first
// found) becomes 'T' before the text is parsed as UTF-8 bytes.
func sanitizeSeparator(text string) string {
	runes := []rune(text)
	for _, position := range [3]int{7, 8, 10} {
		if position >= len(runes) {
			break
		}
		if runes[position] > 0x7f {
			runes[position] = 'T'
			return string(runes)
		}
	}
	return text
}

// isoText is the UTF-8 text; at() returns 0 past the end like the C string terminator.
type isoText string

func (s isoText) at(i int) byte {
	if i >= 0 && i < len(s) {
		return s[i]
	}
	return 0
}

func (s isoText) digits(pos, count int) (int, int, bool) {
	value := 0
	for i := 0; i < count; i++ {
		c := s.at(pos + i)
		if c < '0' || c > '9' {
			return 0, pos, false
		}
		value = value*10 + int(c-'0')
	}
	return value, pos + count, true
}

// separatorLocation is _find_isoformat_datetime_separator.
func (s isoText) separatorLocation() int {
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
	for ; index < n && isDigitByte(s.at(index)); index++ {
	}
	if index < 9 {
		return index
	}
	if index%2 == 0 {
		return 7
	}
	return 8
}

// parseDate is parse_isoformat_date over the first length bytes.
func (s isoText) parseDate(length int, d *DateTime) bool {
	if length < 0 {
		return false
	}
	year, p, ok := s.digits(0, 4)
	if !ok {
		return false
	}
	separated := s.at(p) == '-'
	if separated {
		p++
	}
	if s.at(p) == 'W' {
		week, day := 0, 1
		if week, p, ok = s.digits(p+1, 2); !ok {
			return false
		}
		if p < length {
			if separated {
				if s.at(p) != '-' {
					return false
				}
				p++
			}
			if day, _, ok = s.digits(p, 1); !ok {
				return false
			}
		}
		return isoWeekDate(year, week, day, d)
	}
	month, p, ok := s.digits(p, 2)
	if !ok {
		return false
	}
	if separated {
		if s.at(p) != '-' {
			return false
		}
		p++
	}
	day, _, ok := s.digits(p, 2)
	d.Year, d.Month, d.Day = year, month, day
	return ok
}

// isoWeekDate is iso_to_ymd: week 53 only in long ISO years, weekday 1..7.
func isoWeekDate(year, week, day int, d *DateTime) bool {
	if year < 1 || year > 9999 {
		return false
	}
	if week <= 0 || week >= 53 {
		jan1 := int(time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC).Weekday()+6) % 7
		if week != 53 || !(jan1 == 3 || (jan1 == 2 && isLeap(year))) {
			return false
		}
	}
	if day <= 0 || day >= 8 {
		return false
	}
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)
	monday := jan4.AddDate(0, 0, -int((jan4.Weekday()+6)%7))
	result := monday.AddDate(0, 0, (week-1)*7+day-1)
	d.Year, d.Month, d.Day = result.Year(), int(result.Month()), result.Day()
	return true
}

func isLeap(year int) bool { return year%4 == 0 && (year%100 != 0 || year%400 == 0) }

var monthDays = [13]int{0, 31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

// checkDateTime raises the timezone and datetime constructors' range errors in CPython's
// order (the offset is checked first because the timezone is built first).
func checkDateTime(d DateTime) error {
	if d.Aware && (d.Offset <= -24*time.Hour || d.Offset >= 24*time.Hour) {
		return apperr.ValueError("offset must be a timedelta strictly between -timedelta(hours=24) " +
			"and timedelta(hours=24), not " + TimedeltaRepr(d.Offset) + ".")
	}
	switch {
	case d.Year < 1 || d.Year > 9999:
		return apperr.ValueError(fmt.Sprintf("year %d is out of range", d.Year))
	case d.Month < 1 || d.Month > 12:
		return apperr.ValueError("month must be in 1..12")
	}
	days := monthDays[d.Month]
	if d.Month == 2 && isLeap(d.Year) {
		days = 29
	}
	switch {
	case d.Day < 1 || d.Day > days:
		return apperr.ValueError("day is out of range for month")
	case d.Hour > 23:
		return apperr.ValueError("hour must be in 0..23")
	case d.Minute > 59:
		return apperr.ValueError("minute must be in 0..59")
	case d.Second > 59:
		return apperr.ValueError("second must be in 0..59")
	}
	return nil
}
