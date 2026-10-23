package signalindex

import "studio-engine/internal/pyjson"

const kindAT = uint8(11) // kindCodes["AT"]

// scanned is the result of one pass over the candidates: the matching positions in model order
// and, when asked, the counts per kind (every filter but point_kind) and per logical node (every
// filter but ln), so chips and the object list never need a second pass (ADR-0113).
type scanned struct {
	matches []int
	codes   []uint8 // kind code of every signal; nil when the query needs no kinds
	kinds   [len(kindNames)]int64
	lns     []int64  // count per logical node, when ln_counts=1
	lnNames []string // the logical nodes of lns
}

// scan applies every filter of q. The point kind and the logical node are applied last, so a
// signal that passes all other filters is counted for the chips and the object list.
func (ix *Index) scan(q Query, report *memberSet, values ValueReader) scanned {
	var out scanned
	if q.needsKinds() {
		out.codes = ix.kinds(q.Classes)
	}
	var nodes *nodeTable
	if q.LN != "" || q.LNCounts {
		nodes = ix.logicalNodes()
		out.lnNames = nodes.names
	}
	if q.LNCounts {
		out.lns = make([]int64, len(nodes.names))
	}
	wantKind, kindKnown := kindCodes[q.PointKind]
	var wantLN int32
	lnKnown := false
	if q.LN != "" {
		wantLN, lnKnown = nodes.index[q.LN]
	}
	candidates, all := ix.candidates(q, report)
	if all && q.noFilter() && !q.KindCounts && !q.LNCounts {
		out.matches = make([]int, len(ix.signals))
		for index := range out.matches {
			out.matches[index] = index
		}
		return out
	}
	check := func(position int) {
		item := &ix.signals[position]
		if !ix.keep(q, report, item) || (q.BadQuality && !badQuality(values, item.qualityRef)) {
			return
		}
		var code uint8
		if out.codes != nil {
			code = out.codes[position]
		}
		if q.HideHelpers && code == kindAT {
			return
		}
		kindOK := q.PointKind == "" || (kindKnown && code == wantKind)
		lnOK := q.LN == "" || (lnKnown && nodes.ids[position] == wantLN)
		if q.KindCounts && lnOK {
			out.kinds[code]++
		}
		if q.LNCounts && kindOK {
			out.lns[nodes.ids[position]]++
		}
		if kindOK && lnOK {
			out.matches = append(out.matches, position)
		}
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
	return out
}

// kindCounts is {kind: n} in display order, kinds without signals left out.
func (s *scanned) kindCounts() *pyjson.Object {
	result := pyjson.NewObject()
	for code, count := range s.kinds {
		if count > 0 {
			result.Set(kindNames[code], count)
		}
	}
	return result
}

// lnCounts is {"LD/LN": n}, logical nodes without signals left out (the JSON keys are sorted,
// as in every Flask-shaped answer).
func (s *scanned) lnCounts() *pyjson.Object {
	result := pyjson.NewObject()
	for position, count := range s.lns {
		if count > 0 {
			result.Set(s.lnNames[position], count)
		}
	}
	return result
}
