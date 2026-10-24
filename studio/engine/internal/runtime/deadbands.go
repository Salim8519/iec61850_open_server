package runtime

import (
	"math"
	"strings"

	"studio-engine/internal/runtime/pyval"
)

// measured are the measurement value endings and the instantaneous value beside them
// (IEC 61850-7-3 MV and CMV): mag is reported, instMag/instCVal is always current.
var measured = []struct{ value, instant string }{
	{".cVal.mag.f", ".instCVal.mag.f"}, {".cVal.mag.i", ".instCVal.mag.i"},
	{".mag.f", ".instMag.f"}, {".mag.i", ".instMag.i"},
}

// deadband is the SCD deadband of one measurement value (ADR-0118). It applies only when the
// IED's "Use SCD deadbands" switch is on and the measurement has a rangeC with max > min.
type deadband struct {
	instant string  // the instMag/instCVal reference; "" when the model has none
	band    float64 // db as a value: db/100000 of the range
	zero    float64 // zeroDb as a value: |value| at or below it reports 0
}

// deadbandOf returns the measurement's deadband, or false when none applies.
func deadbandOf(target *running, ref string) (deadband, bool) {
	if !target.bands.Load() {
		return deadband{}, false
	}
	for _, kind := range measured {
		object, found := strings.CutSuffix(ref, kind.value)
		if !found {
			continue
		}
		low, okLow := number(target, object+".rangeC.min.f", object+".rangeC.min.i")
		high, okHigh := number(target, object+".rangeC.max.f", object+".rangeC.max.i")
		if !okLow || !okHigh || high <= low {
			return deadband{}, false
		}
		db, _ := number(target, object+".db")
		zeroDb, _ := number(target, object+".zeroDb")
		result := deadband{band: db / 100000 * (high - low), zero: zeroDb / 100000 * (high - low)}
		if _, has := target.model.index[object+kind.instant]; has {
			result.instant = object + kind.instant
		}
		return result, true
	}
	return deadband{}, false
}

// number is the first of the references the IED has, as a number.
func number(target *running, refs ...string) (float64, bool) {
	for _, ref := range refs {
		if value, has := target.table.get(ref); has {
			return pyval.Float(value)
		}
	}
	return 0, false
}

// filter applies zeroDb and db to a generator value: it returns the value to report and
// whether mag changes. The comparison is with the value mag holds now, whoever wrote it.
func (d deadband) filter(target *running, ref string, value any) (any, bool) {
	current, ok := pyval.Float(value)
	if !ok {
		return value, true
	}
	if d.zero > 0 && math.Abs(current) <= d.zero {
		current = 0
		if strings.HasSuffix(ref, ".i") {
			value = int64(0)
		} else {
			value = 0.0
		}
	}
	last, has := number(target, ref)
	return value, !has || d.band <= 0 || math.Abs(current-last) > d.band
}
