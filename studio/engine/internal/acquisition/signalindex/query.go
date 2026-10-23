package signalindex

import (
	"math"
	"net/url"
	"strings"

	"studio-engine/internal/acquisition/livedom"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/apperr"
)

// Query is one parsed GET /ieds/<id>/signals request, read like Flask's request.args.get
// (the first value of a repeated key).
type Query struct {
	Text       string          // q, casefolded ("" matches everything)
	FC         string          // fc; "" means no filter (`not fc`)
	Kinds      map[string]bool // kind, a comma list without empty items; nil means no filter
	Writable   bool            // writable == "true"
	Report     string          // report reference; "" means no filter
	Ref        string          // exact reference; "" means no filter
	LD         string          // logical device; "" means no filter
	BadQuality bool            // bad_quality == "true"
	InDataset  bool            // in_dataset == "true"
	// Go only (ADR-0095): point_kind (SP, DP, MV …) and helpers == "hide" (no q/t attributes);
	// Classes are the IED's data classes, set by the route when point_kind is asked.
	PointKind   string
	HideHelpers bool
	KindCounts  bool // kind_counts == "1": add {kind: n} counted without point_kind
	Classes     map[string]string
	// Go only (ADR-0113): the smart point picker.
	LN       string // ln: one logical node "LD/LN"; "" means no filter
	LNCounts bool   // ln_counts == "1": add {"LD/LN": n} counted without ln
	Annotate bool   // annotate == "1": each item gets "point_kind"
	Near     string // near: the component's name; "suggested" lists points that mention it
	NearSet  bool   // near was given (even empty): the answer has "suggested"
	offset   intArg
	limit    intArg
}

// intArg is an offset/limit argument; its int() error is reported only after the report
// lookup, in Python's order.
type intArg struct {
	value int64
	err   error
}

// ParseQuery reads the request arguments (use ParseArgs on the raw query string for exact
// Werkzeug parsing). It never fails: an unknown report or a bad offset/limit is reported by
// Search, in the order Python raises them (report first, then offset, then limit).
func ParseQuery(args url.Values) Query {
	get := func(key string) (string, bool) {
		if items := args[key]; len(items) > 0 {
			return items[0], true
		}
		return "", false
	}
	text, _ := get("q")
	fc, _ := get("fc")
	kindText, _ := get("kind")
	writable, _ := get("writable")
	report, _ := get("report")
	ref, _ := get("ref")
	ld, _ := get("ld")
	bad, _ := get("bad_quality")
	inDataset, _ := get("in_dataset")
	pointKind, _ := get("point_kind")
	helpers, _ := get("helpers")
	counts, _ := get("kind_counts")
	ln, _ := get("ln")
	lnCounts, _ := get("ln_counts")
	annotate, _ := get("annotate")
	near, nearSet := get("near")
	query := Query{Text: pyv.Casefold(text), FC: fc, Writable: writable == "true", Report: report, Ref: ref,
		LD: ld, BadQuality: bad == "true", InDataset: inDataset == "true",
		PointKind: pointKind, HideHelpers: helpers == "hide", KindCounts: counts == "1",
		LN: ln, LNCounts: lnCounts == "1", Annotate: annotate == "1", Near: near, NearSet: nearSet}
	for _, kind := range strings.Split(kindText, ",") {
		if kind == "" {
			continue
		}
		if query.Kinds == nil {
			query.Kinds = map[string]bool{}
		}
		query.Kinds[kind] = true
	}
	offset, found := get("offset")
	query.offset = parseIntArg(offset, found, 0)
	limit, found := get("limit")
	query.limit = parseIntArg(limit, found, 100)
	return query
}

// parseIntArg is int(request.args.get(key, fallback)); values beyond int64 saturate, which
// keeps max(0, ...) and min(250, max(1, ...)) exact.
func parseIntArg(text string, found bool, fallback int64) intArg {
	if !found {
		return intArg{value: fallback}
	}
	value, err := livedom.ParseIntText(text)
	if err != nil {
		return intArg{err: apperr.ValueError(err.Error())}
	}
	switch {
	case value.IsInt64():
		return intArg{value: value.Int64()}
	case value.Sign() > 0:
		return intArg{value: math.MaxInt64}
	}
	return intArg{value: math.MinInt64}
}

// window returns the slice bounds [start, end) of the page, or the int() error.
func (q Query) window(total int) (int, int, error) {
	if q.offset.err != nil {
		return 0, 0, q.offset.err
	}
	if q.limit.err != nil {
		return 0, 0, q.limit.err
	}
	offset := max(0, q.offset.value)
	limit := min(250, max(1, q.limit.value))
	start := int(min(offset, int64(total)))
	end := int(min(int64(start)+limit, int64(total)))
	return start, end, nil
}
