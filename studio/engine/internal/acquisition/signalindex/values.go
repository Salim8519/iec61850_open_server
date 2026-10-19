package signalindex

import (
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// ValueReader gives the current value of a data point by reference. found is false when
// the point has no value, which the route shows as 0 (Python:
// index.get(ref, {}).get("value", 0)). A found value may be nil (Python None).
type ValueReader interface {
	Value(ref string) (value any, found bool)
}

// storedValues reads each signal object's own "value" key: what Python answers while the
// IED is stopped. With repeated references the last signal wins, like the route's dict.
type storedValues struct {
	last map[string]*pyjson.Object
}

func (s storedValues) Value(ref string) (any, bool) {
	object := s.last[ref]
	if object == nil {
		return nil, false
	}
	return object.Get("value")
}

// Stored returns the ValueReader over the model's own "value" keys.
func (ix *Index) Stored() ValueReader { return storedValues{last: ix.last} }

// pointValue is index.get(ref, {}).get("value", 0) for a reference value of any type: only
// strings can name a signal.
func pointValue(values ValueReader, ref any) any {
	text, ok := ref.(string)
	if !ok {
		return int64(0)
	}
	if value, found := values.Value(text); found {
		return value
	}
	return int64(0)
}

// badQuality is the bad_quality filter: the value of quality_ref (or of "" when quality_ref
// is falsy) is not in (0, None); 0.0 and False equal 0 like Python's ==.
func badQuality(values ValueReader, qualityRef any) bool {
	if !pyv.Truthy(qualityRef) {
		qualityRef = ""
	}
	value := pointValue(values, qualityRef)
	return value != nil && !pyv.Equal(value, int64(0))
}
