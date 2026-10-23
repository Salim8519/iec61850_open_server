package signalindex

import (
	"strconv"
	"strings"
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// pickerIED is a small bay: a disconnector Q11 with its position, contacts on a GGIO and noise.
func pickerIED() *Index {
	signal := func(ref, fc, kind, description string) any {
		return pyv.O("ref", ref, "fc", fc, "kind", kind, "writable", true, "description", description)
	}
	return Build(pyv.O("id", "bay", "signals", []any{
		signal("BayLD/XSWI1.Pos.stVal", "ST", "Dbpos", "Q11 position"),
		signal("BayLD/XSWI1.Pos.q", "ST", "Quality", ""),
		signal("BayLD/XSWI1.Pos.subVal", "SV", "Dbpos", "Q11 substituted"),
		signal("BayLD/XSWI1.Pos.Oper.ctlVal", "CO", "BOOLEAN", ""),
		signal("BayLD/GGIO3.Q11.stVal", "ST", "BOOLEAN", ""),
		signal("BayLD/GGIO1.Ind1.stVal", "ST", "BOOLEAN", "Q11 open"),
		signal("BayLD/GGIO1.Ind2.stVal", "ST", "BOOLEAN", "q11 closed"),
		signal("BayLD/GGIO1.Ind3.stVal", "ST", "BOOLEAN", "Q110 open"),
		signal("BayLD/GGIO1.Ind106.stVal", "ST", "BOOLEAN", ""),
		signal("BayLD/Q11GGIO2.Ind1.stVal", "ST", "BOOLEAN", "Spare"),
		signal("BayLD/MMXU1.Hz.mag.f", "MX", "FLOAT32", "Frequency"),
	}, "datasets", []any{}, "reports", []any{}))
}

var pickerClasses = map[string]string{"BayLD/XSWI1.Pos": "DPC"}

func search(t *testing.T, ix *Index, args string) *pyjson.Object {
	t.Helper()
	query := ParseQuery(ParseArgs(args))
	query.Classes = pickerClasses
	result, err := ix.Search(query, nil)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func listed(result *pyjson.Object, key, field string) string {
	var parts []string
	for _, item := range pyv.Objects(pyv.GetList(result, key)) {
		parts = append(parts, pyv.GetStr(item, field))
	}
	return strings.Join(parts, " ")
}

func flask(t *testing.T, value any) string {
	t.Helper()
	body, err := pyjson.Flask(value)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(body))
}

// ADR-0113: annotate=1 gives every item Go's kind; the kind of a command comes from its class.
func TestAnnotateGivesEachItemItsKind(t *testing.T) {
	result := search(t, pickerIED(), "annotate=1&limit=4")
	if got := listed(result, "items", "point_kind"); got != "DP AT CF DC" {
		t.Fatalf("kinds %q", got)
	}
}

// ADR-0113: ln filters one logical node; ln_counts counts every node without ln but with every
// other filter; kind_counts keeps ln.
func TestLogicalNodeFilterAndCounts(t *testing.T) {
	result := search(t, pickerIED(), "ln=BayLD/GGIO1&point_kind=SP&ln_counts=1&kind_counts=1")
	if got := listed(result, "items", "ref"); got != "BayLD/GGIO1.Ind1.stVal BayLD/GGIO1.Ind2.stVal "+
		"BayLD/GGIO1.Ind3.stVal BayLD/GGIO1.Ind106.stVal" {
		t.Fatalf("items %q", got)
	}
	if got := flask(t, pyv.Get(result, "ln_counts")); got != `{"BayLD/GGIO1":4,"BayLD/GGIO3":1,"BayLD/Q11GGIO2":1}` {
		t.Fatalf("ln_counts %s", got)
	}
	if got := flask(t, pyv.Get(result, "kind_counts")); got != `{"SP":4}` {
		t.Fatalf("kind_counts %s", got)
	}
	if total := pyv.Get(search(t, pickerIED(), "ln=Nope/GGIO1"), "total"); total != int64(0) {
		t.Fatalf("an unknown node matches nothing: %v", total)
	}
}

// ADR-0113: near lists the points that mention the name as a whole word, descriptions first,
// with every other filter applied and paging ignored.
func TestNearSuggestsPointsThatMentionTheName(t *testing.T) {
	result := search(t, pickerIED(), "near=Q11&point_kind=SP&limit=1&offset=3")
	if got := listed(result, "suggested", "ref"); got != "BayLD/GGIO1.Ind1.stVal BayLD/GGIO1.Ind2.stVal BayLD/GGIO3.Q11.stVal" {
		t.Fatalf("suggested %q", got)
	}
	if got := listed(result, "suggested", "suggested_by"); got != "description description reference" {
		t.Fatalf("suggested_by %q", got)
	}
	if got := listed(search(t, pickerIED(), "near=+q11+"), "suggested", "ref"); !strings.HasPrefix(got,
		"BayLD/XSWI1.Pos.stVal BayLD/XSWI1.Pos.subVal BayLD/GGIO1.Ind1.stVal") {
		t.Fatalf("every kind, ignoring case and spaces: %q", got)
	}
	for _, args := range []string{"near=", "near=--", "near=Q1"} {
		if got := pyv.GetList(search(t, pickerIED(), args), "suggested"); got == nil || len(got) != 0 {
			t.Fatalf("%s: %v", args, got)
		}
	}
	if pyv.Has(search(t, pickerIED(), "q=q11"), "suggested") {
		t.Fatal("no near, no suggested key")
	}
}

func TestNearSuggestsAtMostEight(t *testing.T) {
	signals := []any{}
	for index := range 12 {
		signals = append(signals, pyv.O("ref", "LD/T1GGIO1.Ind"+strconv.Itoa(index)+".stVal", "fc", "ST", "kind",
			"BOOLEAN", "description", "Spare "+strconv.Itoa(index)))
	}
	signals = append(signals, pyv.O("ref", "LD/GGIO9.Ind1.stVal", "fc", "ST", "kind", "BOOLEAN", "description", "T1 trip"))
	ix := Build(pyv.O("id", "x", "signals", signals, "datasets", []any{}, "reports", []any{}))
	if got := listed(search(t, ix, "near=T1"), "suggested", "ref"); got != "LD/GGIO9.Ind1.stVal" {
		t.Fatalf("a glued name is no mention: %q", got)
	}
	result := search(t, ix, "near=spare")
	if got := len(pyv.GetList(result, "suggested")); got != suggestionLimit {
		t.Fatalf("%d suggestions", got)
	}
}

// Requests without the new parameters are answered exactly as before; each new parameter only
// adds its own key (the Python golden test checks the rest).
func TestPickerParametersOnlyAddTheirKeys(t *testing.T) {
	ix := pickerIED()
	for _, base := range []string{"", "q=ggio1", "point_kind=SP&kind_counts=1", "helpers=hide&fc=ST&limit=3"} {
		plain := search(t, ix, base)
		for extra, key := range map[string]string{"ln_counts=1": "ln_counts", "near=Q11": "suggested"} {
			with := search(t, ix, base+"&"+extra)
			with.Delete(key)
			if flask(t, with) != flask(t, plain) {
				t.Fatalf("%s&%s changed the answer", base, extra)
			}
		}
		annotated := search(t, ix, base+"&annotate=1")
		for _, item := range pyv.Objects(pyv.GetList(annotated, "items")) {
			item.Delete("point_kind")
		}
		if flask(t, annotated) != flask(t, plain) {
			t.Fatalf("%s&annotate=1 changed the answer", base)
		}
	}
}

// Kinds are computed once per index and data-class version, and again for a new version.
func TestKindsAreCachedPerDataClassVersion(t *testing.T) {
	ix := pickerIED()
	first := ix.kinds(pickerClasses)
	if &ix.kinds(pickerClasses)[0] != &first[0] {
		t.Fatal("the same classes must reuse the kinds")
	}
	other := map[string]string{"BayLD/XSWI1.Pos": "SPC"}
	if kindNames[ix.kinds(other)[3]] != "SC" || kindNames[ix.kinds(nil)[3]] != "CM" || kindNames[first[3]] != "DC" {
		t.Fatal("a new data-class version must get its own kinds")
	}
}
