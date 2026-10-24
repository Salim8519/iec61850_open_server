// Package deadband describes the SCD deadbands of one IED's measurements for the IED page
// (ADR-0118): the db and zeroDb the SCD gives, the measuring range, the band they make and,
// in plain words, whether ScadaBench applies them. The runtime applies them (deadbands.go).
package deadband

import (
	"sort"
	"strconv"
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// endings are the reported measurement values of MV and CMV data objects.
var endings = []string{".cVal.mag.f", ".cVal.mag.i", ".mag.f", ".mag.i"}

// Points lists every measurement of the IED's signals that has a db, in reference order.
// writable are the db/zeroDb references a SCADA may change (valKind Set or unmarked).
func Points(ied *pyjson.Object, writable []string, enabled bool) *pyjson.Object {
	values := map[string]any{}
	var measured []string
	for _, item := range pyval.List(ied, "signals") {
		object, ok := item.(*pyjson.Object)
		if !ok {
			continue
		}
		ref := pyval.Str(object, "ref")
		values[ref] = pyval.Get(object, "value")
		for _, ending := range endings {
			if strings.HasSuffix(ref, ending) {
				measured = append(measured, ref)
				break
			}
		}
	}
	allowed := map[string]bool{}
	for _, ref := range writable {
		allowed[ref] = true
	}
	sort.Strings(measured)
	points, applied, missing := []any{}, 0, 0
	for _, ref := range measured {
		object := objectOf(ref)
		db, hasDb := pyval.Float(values[object+".db"])
		if _, present := values[object+".db"]; !present {
			continue
		}
		zero, _ := pyval.Float(values[object+".zeroDb"])
		low, okLow := first(values, object+".rangeC.min.f", object+".rangeC.min.i")
		high, okHigh := first(values, object+".rangeC.max.f", object+".rangeC.max.i")
		point := pyval.Obj("ref", ref, "object", object, "db", values[object+".db"], "zero_db", values[object+".zeroDb"],
			"db_writable", allowed[object+".db"], "zero_db_writable", allowed[object+".zeroDb"])
		ranged := okLow && okHigh && high > low
		if ranged {
			point.Set("range_min", low).Set("range_max", high).
				Set("band", db/100000*(high-low)).Set("zero_band", zero/100000*(high-low))
		} else {
			missing++
		}
		applies := enabled && ranged && hasDb
		if applies {
			applied++
		}
		point.Set("applies", applies).Set("note", note(ranged, db, zero, low, high))
		points = append(points, point)
	}
	return pyval.Obj("enabled", enabled, "points", points, "applied", int64(applied), "without_range", int64(missing))
}

// objectOf is the data object of a measurement value (the reference without mag/cVal).
func objectOf(ref string) string {
	for _, ending := range endings {
		if object, found := strings.CutSuffix(ref, ending); found {
			return object
		}
	}
	return ref
}

func first(values map[string]any, refs ...string) (float64, bool) {
	for _, ref := range refs {
		if value, present := values[ref]; present {
			return pyval.Float(value)
		}
	}
	return 0, false
}

// note says in plain words what the deadband does when the switch is on.
func note(ranged bool, db, zero, low, high float64) string {
	if !ranged {
		return "No measuring range (rangeC) in the SCD, so no deadband: every change is reported."
	}
	span := high - low
	text := "Reported when the value moves more than " + number(db/100000*span) + " (db " + number(db) +
		" = " + number(db/1000) + " % of the range " + number(low) + " to " + number(high) + ")."
	if db == 0 {
		text = "db is 0: every change is reported."
	}
	if zero > 0 {
		text += " Values within " + number(zero/100000*span) + " of zero are reported as 0."
	}
	if db >= 100000 {
		text += " This band is the whole range, so the reported value (mag) almost never changes; instMag always has the live value."
	}
	return text
}

// number is a short plain number: 6 significant digits, never an exponent (1000000, not 1e+06).
func number(value float64) string {
	short, _ := strconv.ParseFloat(strconv.FormatFloat(value, 'g', 6, 64), 64)
	return strconv.FormatFloat(short, 'f', -1, 64)
}
