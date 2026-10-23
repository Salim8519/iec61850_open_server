package runtime

import (
	"math"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Generator samples arrive ten times a second per source and made the run log grow by
// megabytes a minute (ADR-0115). The bus still carries every sample for the live view; the
// run log keeps one signal_summary per IED and second instead, with the bus ids of the first
// and last sample it covers (the correlation back to the live stream) and each point's last
// applied value. Every other event, manual and scenario writes included, stays one line.

// summarised reports whether a bus event is a generator sample the log folds into a summary.
func summarised(event *pyjson.Object) bool {
	return pyval.Get(event, "kind") == "signal" && pyval.Get(event, "source") == "generator"
}

type window struct {
	ied          string
	second       int64
	samples      int64
	fromID, toID any
	values       *pyjson.Object
}

// summaries holds the open windows in first-sample order (a deterministic log).
type summaries struct {
	open  []*window
	byIed map[string]*window
}

func newSummaries() *summaries { return &summaries{byIed: map[string]*window{}} }

func eventSecond(event *pyjson.Object) (int64, bool) {
	at, ok := pyval.Float(pyval.Get(event, "timestamp"))
	if !ok {
		return 0, false
	}
	return int64(math.Floor(at)), true
}

// add folds one generator sample into its IED's window for that second.
func (s *summaries) add(event *pyjson.Object, second int64) {
	ied := pyval.Str(event, "ied_id")
	current := s.byIed[ied]
	if current == nil || current.second != second {
		current = &window{ied: ied, second: second, fromID: pyval.Get(event, "id"), values: pyjson.NewObject()}
		s.byIed[ied] = current
		s.open = append(s.open, current)
	}
	current.samples++
	current.toID = pyval.Get(event, "id")
	current.values.Set(pyval.Str(event, "ref"), pyval.Get(event, "value"))
}

// due returns the windows of seconds before `before` (all when flushAll), oldest first, and
// forgets them.
func (s *summaries) due(before int64, flushAll bool) []*pyjson.Object {
	var records []*pyjson.Object
	kept := s.open[:0]
	for _, item := range s.open {
		if !flushAll && item.second >= before {
			kept = append(kept, item)
			continue
		}
		if s.byIed[item.ied] == item {
			delete(s.byIed, item.ied)
		}
		records = append(records, pyval.Obj("kind", "signal_summary", "ied_id", item.ied, "source", "generator",
			"second", item.second, "samples", item.samples, "from_id", item.fromID, "to_id", item.toID,
			"values", item.values))
	}
	s.open = kept
	return records
}
