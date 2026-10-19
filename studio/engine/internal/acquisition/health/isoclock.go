package health

import "time"

// parseTime is parse_isoformat_time from index start: hour, minute, second, microsecond,
// the UTC offset, whether an offset was given, and whether the text parsed. Like CPython,
// characters between the time and an offset are tolerated, and a "Z" ends the text.
func parseTime(s isoText, start int) ([4]int, time.Duration, bool, bool) {
	end := len(s)
	zone := start
	for {
		c := s.at(zone)
		if c == 'Z' || c == '+' || c == '-' {
			break
		}
		zone++
		if zone >= end {
			break
		}
	}
	clock, code := clockFields(s, start, zone)
	if code < 0 {
		return clock, 0, false, false
	}
	if zone == end {
		return clock, 0, false, code == 0
	}
	if s.at(zone) == 'Z' {
		return clock, 0, true, s.at(zone+1) == 0
	}
	sign := time.Duration(1)
	if s.at(zone) == '-' {
		sign = -1
	}
	offset, code := clockFields(s, zone+1, end)
	if code != 0 {
		return clock, 0, false, false
	}
	seconds := time.Duration(offset[0]*3600+offset[1]*60+offset[2]) * time.Second
	if seconds == 0 {
		return clock, 0, true, true // CPython uses UTC when the whole seconds are 0, dropping microseconds
	}
	return clock, sign * (seconds + time.Duration(offset[3])*time.Microsecond), true, true
}

// clockFields is parse_hh_mm_ss_ff over s[start:end]: [HH[:?MM[:?SS]]][.,fraction] as
// hour, minute, second and microsecond (extra fraction digits are truncated). code is 0
// at a clean end, 1 when other characters follow and -1 for malformed text.
func clockFields(s isoText, start, end int) ([4]int, int) {
	var fields [4]int
	p := start
	separated := true
	for i := 0; i < 3; i++ {
		value, next, ok := s.digits(p, 2)
		if !ok {
			return fields, -1
		}
		fields[i] = value
		c := s.at(next)
		p = next + 1
		if i == 0 {
			separated = c == ':'
		}
		if p >= end {
			if c != 0 {
				return fields, 1
			}
			return fields, 0
		}
		if separated && c == ':' {
			continue
		}
		if c == '.' || c == ',' {
			break
		}
		if separated {
			return fields, -1
		}
		p--
	}
	count := min(end-p, 6)
	fraction, next, ok := s.digits(p, count)
	if !ok {
		return fields, -1
	}
	for ; count < 6; count++ {
		fraction *= 10
	}
	fields[3] = fraction
	for isDigit(s.at(next)) {
		next++
	}
	if s.at(next) != 0 {
		return fields, 1
	}
	return fields, 0
}
