package pyrules

import (
	"strconv"

	"studio-engine/internal/apperr"
)

// Proleptic Gregorian calendar rules of CPython datetime.

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
