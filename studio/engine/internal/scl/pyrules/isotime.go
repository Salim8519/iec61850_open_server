package pyrules

import (
	"math"
	"math/big"
	"strconv"

	"studio-engine/internal/apperr"
)

// ISOTimestampMillis is int(datetime.fromisoformat(text.replace("Z", "+00:00")) as UTC
// when naive, .timestamp() * 1000) — the Timestamp initial value of type_resolver.py —
// following CPython 3.12's C fromisoformat parser byte for byte.
func ISOTimestampMillis(text string) (int64, error) {
	text = replaceZ(text)
	invalid := apperr.ValueError("Invalid isoformat string: " + Repr(text))
	if len([]rune(text)) < 7 {
		return 0, invalid
	}
	sep := separatorLocation(text)
	if sep < 0 || sep > len(text) {
		return 0, invalid
	}
	var t isoParts
	ok := parseISODate(text, sep, &t)
	withZone := false
	if ok && len(text) > sep {
		rest := text[sep+1:]
		var rv int
		rv, ok = parseISOTime(rest, &t)
		withZone = rv == 1
	}
	if !ok {
		return 0, invalid
	}
	if err := checkRanges(t); err != nil {
		return 0, err
	}
	offset := int64(0)
	if withZone {
		offset = int64(t.tzSeconds)*1000000 + int64(t.tzMicros)
		if offset <= -86400000000 || offset >= 86400000000 {
			return 0, apperr.ValueError("offset must be a timedelta strictly between -timedelta(hours=24) and timedelta(hours=24).")
		}
	}
	days := ordinal(t.year, t.month, t.day) - ordinal(1970, 1, 1)
	micros := (int64(days)*86400+int64(t.hour)*3600+int64(t.minute)*60+int64(t.second))*1000000 +
		int64(t.micro) - offset
	seconds, _ := new(big.Rat).SetFrac64(micros, 1000000).Float64() // int / int true division
	return int64(math.Trunc(seconds * 1000)), nil
}

func replaceZ(text string) string {
	out := []byte{}
	for i := 0; i < len(text); i++ {
		if text[i] == 'Z' {
			out = append(out, "+00:00"...)
		} else {
			out = append(out, text[i])
		}
	}
	return string(out)
}

type isoParts struct {
	year, month, day, hour, minute, second, micro, tzSeconds, tzMicros int
}

func at(s string, i int) byte {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func isDigitByte(c byte) bool { return c >= '0' && c <= '9' }

// digits is parse_digits: exactly n digits at s[p:], or -1.
func digits(s string, p, n int, out *int) int {
	for i := 0; i < n; i++ {
		c := at(s, p)
		if !isDigitByte(c) {
			return -1
		}
		*out = *out*10 + int(c-'0')
		p++
	}
	return p
}

func separatorLocation(s string) int {
	if len(s) == 7 {
		return 7
	}
	if at(s, 4) == '-' {
		if at(s, 5) != 'W' {
			return 10
		}
		if len(s) < 8 {
			return -1
		}
		if len(s) > 8 && at(s, 8) == '-' {
			if len(s) == 9 {
				return -1
			}
			if len(s) > 10 && isDigitByte(at(s, 10)) {
				return 8
			}
			return 10
		}
		return 8
	}
	if at(s, 4) == 'W' {
		idx := 7
		for idx < len(s) && isDigitByte(s[idx]) {
			idx++
		}
		if idx < 9 {
			return idx
		}
		if idx%2 == 0 {
			return 7
		}
		return 8
	}
	return 8
}

func parseISODate(s string, length int, t *isoParts) bool {
	p := digits(s, 0, 4, &t.year)
	if p < 0 {
		return false
	}
	separated := at(s, p) == '-'
	if separated {
		p++
	}
	if at(s, p) == 'W' {
		p++
		week, day := 0, 0
		if p = digits(s, p, 2, &week); p < 0 {
			return false
		}
		if p < length {
			if separated {
				if at(s, p) != '-' {
					return false
				}
				p++
			}
			if p = digits(s, p, 1, &day); p < 0 {
				return false
			}
		} else {
			day = 1
		}
		return isoToYMD(t, week, day)
	}
	if p = digits(s, p, 2, &t.month); p < 0 {
		return false
	}
	if separated {
		if at(s, p) != '-' {
			return false
		}
		p++
	}
	return digits(s, p, 2, &t.day) >= 0
}

// parseISOTime is parse_isoformat_time; rv 1 means a time zone was given.
func parseISOTime(s string, t *isoParts) (int, bool) {
	end := len(s)
	zone := 0
	for zone < end && s[zone] != 'Z' && s[zone] != '+' && s[zone] != '-' {
		zone++
	}
	if end == 0 {
		return 0, false
	}
	rv := hhmmssff(s, 0, zone, &t.hour, &t.minute, &t.second, &t.micro)
	if rv < 0 {
		return 0, false
	}
	if zone == end {
		return 0, rv != 1
	}
	if s[zone] == 'Z' {
		return 1, zone+1 == end
	}
	sign := 1
	if s[zone] == '-' {
		sign = -1
	}
	hour, minute, second := 0, 0, 0
	rv = hhmmssff(s, zone+1, end, &hour, &minute, &second, &t.tzMicros)
	t.tzSeconds = sign * (hour*3600 + minute*60 + second)
	t.tzMicros *= sign
	return 1, rv == 0
}

func hhmmssff(s string, p, end int, hour, minute, second, micro *int) int {
	values := []*int{hour, minute, second}
	separated := true
	for i := 0; i < 3; i++ {
		if p = digits(s, p, 2, values[i]); p < 0 {
			return -3
		}
		c := at(s, p)
		p++
		if i == 0 {
			separated = c == ':'
		}
		switch {
		case p >= end:
			if c != 0 && p-1 < len(s) {
				return 1
			}
			return 0
		case separated && c == ':':
			continue
		case c == '.' || c == ',':
			i = 3
			continue
		case !separated:
			p--
		default:
			return -4
		}
	}
	remains := end - p
	count := remains
	if count >= 6 {
		count = 6
	}
	if count <= 0 {
		return -3
	}
	if p = digits(s, p, count, micro); p < 0 {
		return -3
	}
	if count < 6 {
		*micro *= []int{100000, 10000, 1000, 100, 10}[count-1]
	}
	for isDigitByte(at(s, p)) {
		p++
	}
	if at(s, p) != 0 {
		return 1
	}
	return 0
}

func isLeap(year int) bool { return year%4 == 0 && (year%100 != 0 || year%400 == 0) }

func daysInMonth(year, month int) int {
	days := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}[month-1]
	if month == 2 && isLeap(year) {
		days++
	}
	return days
}

// ordinal is the proleptic Gregorian ordinal (0001-01-01 = 1).
func ordinal(year, month, day int) int {
	y := year - 1
	days := y*365 + y/4 - y/100 + y/400
	for m := 1; m < month; m++ {
		days += daysInMonth(year, m)
	}
	return days + day
}

func isoToYMD(t *isoParts, week, day int) bool {
	if week <= 0 || week >= 53 {
		first := (ordinal(t.year, 1, 1) + 6) % 7
		if !(week == 53 && (first == 3 || first == 2 && isLeap(t.year))) {
			return false
		}
	}
	if day <= 0 || day >= 8 {
		return false
	}
	first := ordinal(t.year, 1, 1)
	weekday := (first + 6) % 7
	monday := first - weekday
	if weekday > 3 {
		monday += 7
	}
	target := monday + (week-1)*7 + day - 1
	t.year, t.month, t.day = fromOrdinal(target)
	return true
}

func fromOrdinal(n int) (int, int, int) {
	year := 1
	for ordinal(year+400, 1, 1) <= n {
		year += 400
	}
	for ordinal(year+1, 1, 1) <= n {
		year++
	}
	month := 1
	for month < 12 && ordinal(year, month+1, 1) <= n {
		month++
	}
	return year, month, n - ordinal(year, month, 1) + 1
}

func checkRanges(t isoParts) error {
	switch {
	case t.year < 1 || t.year > 9999:
		return apperr.ValueError("year " + strconv.Itoa(t.year) + " is out of range")
	case t.month < 1 || t.month > 12:
		return apperr.ValueError("month must be in 1..12")
	case t.day < 1 || t.day > daysInMonth(t.year, t.month):
		return apperr.ValueError("day is out of range for month")
	case t.hour > 23:
		return apperr.ValueError("hour must be in 0..23")
	case t.minute > 59:
		return apperr.ValueError("minute must be in 0..59")
	case t.second > 59:
		return apperr.ValueError("second must be in 0..59")
	}
	return nil
}
