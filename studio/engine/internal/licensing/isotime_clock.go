package licensing

import (
	"math"
	"math/big"
	"time"
)

var fractionScale = [...]int{100000, 10000, 1000, 100, 10}

// hhmmss is parse_hh_mm_ss_ff over s[start:end]. Like the C code it reads the byte after
// each component even past end (NUL at the very end) and returns 1 when more text
// follows; a negative result is an error.
func (s isoText) hhmmss(start, end int) (values [4]int, rv int) {
	p, separated := start, true
	for index := 0; index < 3; index++ {
		value, next, ok := s.digits(p, 2)
		if !ok {
			return values, -3
		}
		values[index] = value
		c := s.at(next)
		p = next + 1
		if index == 0 {
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
		if !separated {
			p--
			continue
		}
		return values, -4
	}
	count := min(end-p, 6)
	micro, next, ok := s.digits(p, count)
	if !ok {
		return values, -3
	}
	if count < 6 {
		micro *= fractionScale[count-1]
	}
	values[3] = micro
	for isDigitByte(s.at(next)) {
		next++
	}
	if s.at(next) != 0 {
		return values, 1
	}
	return values, 0
}

// clock is parse_isoformat_time on the text after the date separator.
func (s isoText) clock(t *isoTime) bool {
	end, tz := len(s), 0
	for {
		if c := s.at(tz); c == 'Z' || c == '+' || c == '-' {
			break
		}
		tz++
		if tz >= end {
			break
		}
	}
	values, rv := s.hhmmss(0, tz)
	if rv < 0 {
		return false
	}
	t.hour, t.minute, t.second, t.micro = values[0], values[1], values[2], values[3]
	if tz == end {
		return rv != 1
	}
	t.aware = true
	if s.at(tz) == 'Z' {
		return s.at(tz+1) == 0
	}
	sign := 1
	if s.at(tz) == '-' {
		sign = -1
	}
	zone, rv := s.hhmmss(tz+1, end)
	if rv != 0 {
		return false
	}
	t.offsetSeconds = sign * (zone[0]*3600 + zone[1]*60 + zone[2])
	t.offsetMicro = sign * zone[3]
	return true
}

// timestamp is int(datetime.timestamp()): aware times exactly (total microseconds / 10**6,
// correctly rounded, then truncated), naive times in the local time zone.
func (t isoTime) timestamp() int64 {
	if !t.aware {
		local := time.Date(t.year, time.Month(t.month), t.day, t.hour, t.minute, t.second, 0, time.Local)
		return int64(float64(local.Unix()) + float64(t.micro)/1e6)
	}
	utc := time.Date(t.year, time.Month(t.month), t.day, t.hour, t.minute, t.second, 0, time.UTC)
	micros := big.NewInt(utc.Unix())
	micros.Mul(micros, big.NewInt(1_000_000))
	micros.Add(micros, big.NewInt(int64(t.micro)))
	micros.Sub(micros, big.NewInt(int64(t.offsetSeconds)*1_000_000+int64(t.offsetMicro)))
	seconds, _ := new(big.Rat).SetFrac(micros, big.NewInt(1_000_000)).Float64()
	return int64(math.Trunc(seconds))
}
