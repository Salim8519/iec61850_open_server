package service

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Sample is the value one monitoring point carries now (domain/gateway_values.sample).
type Sample struct {
	ID       string
	Value    any
	Quality  any
	TimeMS   any
	SourceOK bool
}

// Object is the {"id", "value", "quality", "time_ms", "source_ok"} sent to studio_gateway.
func (s Sample) Object() *pyjson.Object {
	return pyjson.NewObject().Set("id", s.ID).Set("value", s.Value).Set("quality", s.Quality).
		Set("time_ms", s.TimeMS).Set("source_ok", s.SourceOK)
}

// Same is Python's dict == between two samples.
func (s Sample) Same(other Sample) bool {
	return s.ID == other.ID && s.SourceOK == other.SourceOK && pytext.Equal(s.Value, other.Value) &&
		pytext.Equal(s.Quality, other.Quality) && pytext.Equal(s.TimeMS, other.TimeMS)
}

// IEDOnline is ied_online: simulated IEDs need Run; real IEDs need a live connection.
func IEDOnline(ied *pyjson.Object, runState string) bool {
	if runState != "running" && runState != "paused" {
		return false
	}
	if get(ied, "role") == "real" {
		return get(ied, "state") == "real_live"
	}
	return get(ied, "state") == "running"
}

// SampleOf is sample(point, index, ied, run_state) for the point (iedID, ref).
func SampleOf(id, iedID, ref string, ied *pyjson.Object, runState string, runtime Runtime) Sample {
	signal, found := runtime.Signal(iedID, ref)
	if !found || ied == nil {
		return Sample{ID: id, Value: int64(0), Quality: int64(0), TimeMS: int64(0)}
	}
	value := signal.Value
	if flag, isBool := value.(bool); isBool {
		value = int64(0)
		if flag {
			value = int64(1)
		}
	}
	if !pytext.IsNumber(value) {
		value = int64(0)
	}
	var quality, timeMS any = int64(0), int64(0)
	if signal.QualityRef != "" {
		if record, ok := runtime.Signal(iedID, signal.QualityRef); ok {
			quality = valueOrZero(record.Value)
		}
	}
	if signal.TimestampRef != "" {
		if record, ok := runtime.Signal(iedID, signal.TimestampRef); ok {
			timeMS = valueOrZero(record.Value)
		}
	}
	if !pytext.IsIntOrBool(quality) {
		quality = int64(0)
	}
	fresh := signal.LiveState == "" || signal.LiveState == "fresh"
	return Sample{ID: id, Value: value, Quality: quality, TimeMS: timeMS, SourceOK: IEDOnline(ied, runState) && fresh}
}

func valueOrZero(value any) any {
	if value == nil {
		return int64(0) // record.get("value", 0): an absent value counts as 0
	}
	return value
}

var doubleText = map[int64]string{0: "intermediate", 1: "OFF (open)", 2: "ON (closed)", 3: "bad state"}

// Display is display(point, current): the point table's plain text of a value.
func Display(point *pyjson.Object, current Sample) string {
	if !current.SourceOK {
		return "no value (IED not running or not connected)"
	}
	var text string
	switch get(point, "kind") {
	case "single":
		text = "OFF"
		if pytext.Truthy(current.Value) {
			text = "ON"
		}
	case "double":
		text = pytext.Str(current.Value)
		if key, ok := dictKey(current.Value); ok {
			if word, found := doubleText[key]; found {
				text = word
			}
		}
	default:
		scale, found := point.Get("scale")
		if !found {
			scale = 1.0
		}
		value, _ := pytext.Float(current.Value)
		factor, _ := pytext.Float(scale)
		text = pytext.FormatG(value*factor, 6)
	}
	bits, _ := pytext.Int64(current.Quality)
	switch bits & 3 {
	case 2:
		text += " \xc2\xb7 invalid"
	case 3:
		text += " \xc2\xb7 questionable"
	}
	return text
}

// dictKey is how a number finds an int key of a Python dict (1.0 and True find 1).
func dictKey(value any) (int64, bool) {
	if f, isFloat := value.(float64); isFloat {
		if f == float64(int64(f)) {
			return int64(f), true
		}
		return 0, false
	}
	return pytext.Int64(value)
}
