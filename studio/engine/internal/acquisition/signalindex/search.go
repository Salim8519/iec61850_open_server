package signalindex

import (
	"strings"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// Search answers one query: {"total": n, "items": [signal + "quality" + "timestamp"],
// "devices": [...]}. values gives the current point values; nil reads the model's own
// "value" keys (Stored). Errors come in Python's order: an unknown report
// (StudioError "Unknown report reference"), then int() of offset, then of limit (ValueError).
// The Go-only parameters (ADR-0095, ADR-0113) add keys after these and never change them for a
// request that does not use them.
func (ix *Index) Search(q Query, values ValueReader) (*pyjson.Object, error) {
	if values == nil {
		values = ix.Stored()
	}
	var report *memberSet
	if q.Report != "" {
		set, found := ix.reports[q.Report]
		if !found {
			return nil, apperr.New("Unknown report reference")
		}
		report = &set
	}
	found := ix.scan(q, report, values)
	start, end, err := q.window(len(found.matches))
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, position := range found.matches[start:end] {
		items = append(items, ix.item(position, values, found.codes, q.Annotate))
	}
	devices := pyv.Strings(ix.devices)
	result := pyv.O("total", int64(len(found.matches)), "items", items, "devices", devices)
	if q.KindCounts { // Go only (ADR-0095)
		result.Set("kind_counts", found.kindCounts())
	}
	if q.LNCounts { // Go only (ADR-0113)
		result.Set("ln_counts", found.lnCounts())
	}
	if q.NearSet {
		result.Set("suggested", ix.suggested(q, found, values))
	}
	return result, nil
}

// item is one listed signal: a copy of the model's signal with its current quality and time,
// and its point kind when annotate=1 asked for it.
func (ix *Index) item(position int, values ValueReader, codes []uint8, annotate bool) *pyjson.Object {
	item := &ix.signals[position]
	object := pyv.Shallow(item.object)
	object.Set("quality", pointValue(values, item.qualityRef))
	object.Set("timestamp", pointValue(values, item.timeRef))
	if annotate {
		object.Set("point_kind", kindNames[codes[position]])
	}
	return object
}

// candidates picks the positions to check; all is true when every signal is a candidate.
func (ix *Index) candidates(q Query, report *memberSet) ([]int, bool) {
	switch {
	case q.Ref != "":
		return ix.byRef[q.Ref], false
	case report != nil:
		return report.positions, false
	case q.InDataset:
		return ix.inDataset.positions, false
	case q.Text != "":
		if positions, ok := ix.text.matches(q.Text); ok {
			return positions, false
		}
	}
	return nil, true
}

// noFilter is true when every condition of the route's comprehension holds for any signal.
func (q Query) noFilter() bool {
	return q.Text == "" && q.FC == "" && q.Ref == "" && q.Kinds == nil && !q.Writable && q.Report == "" &&
		q.LD == "" && !q.BadQuality && !q.InDataset && q.PointKind == "" && !q.HideHelpers && q.LN == ""
}

// keep applies the value-independent conditions of the comprehension (the point kind and the
// logical node are checked by scan, which also counts them).
func (ix *Index) keep(q Query, report *memberSet, item *signal) bool {
	switch {
	case q.Text != "" && !strings.Contains(item.haystack, q.Text):
		return false
	case q.FC != "" && !(item.fcOK && item.fc == q.FC):
		return false
	case q.Ref != "" && item.ref != q.Ref:
		return false
	case q.Kinds != nil && !(item.kindOK && q.Kinds[item.kind]):
		return false
	case q.Writable && !item.writable:
		return false
	case report != nil && !report.refs[item.ref]:
		return false
	case q.InDataset && !ix.inDataset.refs[item.ref]:
		return false
	case q.LD != "" && item.ld != q.LD:
		return false
	}
	return true
}
