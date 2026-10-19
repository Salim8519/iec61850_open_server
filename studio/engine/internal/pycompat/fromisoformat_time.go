package pycompat

import "time"

var fractionScale = [6]int{100000, 10000, 1000, 100, 10, 1}

// parseTime is parse_isoformat_time over the rest of the text: the clock up to the first
// 'Z', '+' or '-', then the offset.
func (s isoText) parseTime(d *DateTime) bool {
	n := len(s)
	zone := 0
	for {
		if c := s.at(zone); c == 'Z' || c == '+' || c == '-' {
			break
		}
		if zone++; zone >= n {
			break
		}
	}
	clock, rv := s.clock(0, zone)
	if rv < 0 {
		return false
	}
	d.Hour, d.Minute, d.Second, d.Microsecond = clock[0], clock[1], clock[2], clock[3]
	if zone >= n {
		return rv == 0
	}
	d.Aware = true
	if s.at(zone) == 'Z' {
		return s.at(zone+1) == 0
	}
	sign := 1
	if s.at(zone) == '-' {
		sign = -1
	}
	offset, rv := s.clock(zone+1, n)
	seconds := sign * (offset[0]*3600 + offset[1]*60 + offset[2])
	if seconds != 0 {
		d.Offset = time.Duration(seconds)*time.Second + time.Duration(sign*offset[3])*time.Microsecond
	}
	return rv == 0
}

// clock is parse_hh_mm_ss_ff: HH[:?MM[:?SS]] with the separator style of the first
// component, then a fraction of up to six digits (further digits skipped). It returns
// 1 when text remains before the C terminator, 0 at the end, negative on failure.
func (s isoText) clock(start, end int) ([4]int, int) {
	var values [4]int
	p, separated := start, true
	for i := 0; i < 3; i++ {
		value, next, ok := s.digits(p, 2)
		if !ok {
			return values, -3
		}
		values[i], p = value, next
		c := s.at(p)
		p++
		if i == 0 {
			separated = c == ':'
		}
		if p >= end {
			if c != 0 {
				return values, 1
			}
			return values, 0
		}
		if separated && c == ':' {
			continue
		}
		if c == '.' || c == ',' {
			break
		}
		if separated {
			return values, -4
		}
		p--
	}
	count := end - p
	if count > 6 {
		count = 6
	}
	fraction, next, ok := s.digits(p, count)
	if !ok || count < 1 {
		return values, -3
	}
	values[3] = fraction * fractionScale[count-1]
	for p = next; isDigitByte(s.at(p)); p++ {
	}
	if s.at(p) != 0 {
		return values, 1
	}
	return values, 0
}
