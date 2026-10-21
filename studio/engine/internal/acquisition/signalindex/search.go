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
	matches := ix.filter(q, report, values)
	start, end, err := q.window(len(matches))
	if err != nil {
		return nil, err
	}
	items := make([]any, 0, end-start)
	for _, position := range matches[start:end] {
		item := ix.signals[position]
		object := pyv.Shallow(item.object)
		object.Set("quality", pointValue(values, item.qualityRef))
		object.Set("timestamp", pointValue(values, item.timeRef))
		items = append(items, object)
	}
	devices := pyv.Strings(ix.devices)
	return pyv.O("total", int64(len(matches)), "items", items, "devices", devices), nil
}

// filter returns the positions of the matching signals in model order. It starts from the
// smallest known candidate list (exact reference, report or dataset members, q) instead of
// scanning every signal, and skips the per-signal checks entirely when nothing filters.
func (ix *Index) filter(q Query, report *memberSet, values ValueReader) []int {
	candidates, all := ix.candidates(q, report)
	if all && q.noFilter() {
		positions := make([]int, len(ix.signals))
		for index := range positions {
			positions[index] = index
		}
		return positions
	}
	var matches []int
	check := func(position int) {
		item := &ix.signals[position]
		if !ix.keep(q, report, item) {
			return
		}
		if q.BadQuality && !badQuality(values, item.qualityRef) {
			return
		}
		matches = append(matches, position)
	}
	if all {
		for position := range ix.signals {
			check(position)
		}
	} else {
		for _, position := range candidates {
			check(position)
		}
	}
	return matches
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
		q.LD == "" && !q.BadQuality && !q.InDataset
}

// keep applies the value-independent conditions of the comprehension.
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
