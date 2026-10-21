package signalindex

import (
	"regexp"
	"strings"

	"studio-engine/internal/pyjson"
)

// Point kinds (ADR-0095), the same rules as the frontend's points/pointKind.ts kindOf61850:
// FC + basic type, refined by the data object's class (cdc) when it is known. They back the
// Go-only point_kind and helpers=hide filters of GET /ieds/<id>/signals; a request without them
// is answered exactly as Flask does.

var (
	numericKind = regexp.MustCompile(`^(INT(8|16|32|64|128)U?|FLOAT(32|64))$`)
	textualKind = regexp.MustCompile(`^(VisString|Unicode|Octet)`)
	commandKind = map[string]string{"SPC": "SC", "DPC": "DC", "APC": "SE", "BAC": "SE", "INC": "SE",
		"ENC": "SE", "BSC": "SE", "ISC": "SE"}
)

// pointKind is kindOf61850(fc, basicType, dataClass).
func pointKind(fc, basic, class string) string {
	switch {
	case basic == "Quality" || basic == "Timestamp":
		return "AT"
	case fc == "CO":
		if kind, ok := commandKind[class]; ok {
			return kind
		}
		if numericKind.MatchString(basic) {
			return "SE"
		}
		return "CM"
	case fc == "SP" || fc == "SG" || fc == "SE" || fc == "CF":
		return "CF"
	case fc == "DC" || fc == "EX" || textualKind.MatchString(basic):
		return "TX"
	case basic == "Dbpos":
		return "DP"
	case basic == "BOOLEAN" && fc == "ST":
		return "SP"
	case fc == "MX" && numericKind.MatchString(basic):
		return "MV"
	case fc == "ST" && (basic == "Enum" || numericKind.MatchString(basic)):
		if class == "BCR" {
			return "CT"
		}
		return "IS"
	}
	return "UN"
}

// dataClassOf is the class of the longest object reference that equals ref or is followed by
// "." in it (points/dataClasses.ts dataClassOf).
func dataClassOf(classes map[string]string, ref string) string {
	for at := len(ref); at > 0; at = strings.LastIndex(ref[:at], ".") {
		if class, ok := classes[ref[:at]]; ok {
			return class
		}
	}
	return ""
}

// kindOf is the point kind of one indexed signal for this query's classes.
func (q Query) kindOf(item *signal) string {
	if !item.fcOK || !item.kindOK {
		return "UN"
	}
	return pointKind(item.fc, item.kind, dataClassOf(q.Classes, item.ref))
}

// keepKind applies point_kind and helpers=hide.
func (q Query) keepKind(item *signal) bool {
	if q.PointKind == "" && !q.HideHelpers {
		return true
	}
	kind := q.kindOf(item)
	if q.HideHelpers && kind == "AT" {
		return false
	}
	return q.PointKind == "" || kind == q.PointKind
}

// kindCounts counts the signals of every kind that pass all the other filters (point_kind
// itself left out), so the type chips can hide kinds with no rows.
func (ix *Index) kindCounts(q Query, report *memberSet, values ValueReader) *pyjson.Object {
	q.PointKind = ""
	counts := map[string]int64{}
	for _, position := range ix.filter(q, report, values) {
		counts[q.kindOf(&ix.signals[position])]++
	}
	result := pyjson.NewObject()
	for _, kind := range []string{"SP", "DP", "IS", "MV", "CT", "SC", "DC", "SE", "CM", "CF", "TX", "AT", "UN"} {
		if counts[kind] > 0 {
			result.Set(kind, counts[kind])
		}
	}
	return result
}
