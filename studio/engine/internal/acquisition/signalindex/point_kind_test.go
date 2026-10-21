package signalindex

import (
	"net/url"
	"strings"
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// ADR-0095: point_kind and helpers=hide filter by the same kinds as the frontend.
func TestPointKindFilter(t *testing.T) {
	signal := func(ref, fc, kind string) any {
		return pyv.O("ref", ref, "fc", fc, "kind", kind, "writable", false, "description", "")
	}
	ied := pyv.O("signals", []any{
		signal("IEDLD/XCBR1.Pos.stVal", "ST", "Dbpos"), signal("IEDLD/XCBR1.Pos.q", "ST", "Quality"),
		signal("IEDLD/XCBR1.Pos.Oper.ctlVal", "CO", "BOOLEAN"), signal("IEDLD/PTRC1.Tr.general", "ST", "BOOLEAN"),
		signal("IEDLD/MMTR1.SupWh.actVal", "ST", "INT64"), signal("IEDLD/MMXU1.Hz.mag.f", "MX", "FLOAT32"),
	}, "datasets", []any{}, "reports", []any{})
	ix := Build(ied)
	refs := func(args string, classes map[string]string) []string {
		values, _ := url.ParseQuery(args)
		query := ParseQuery(values)
		query.Classes = classes
		result, err := ix.Search(query, nil)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, item := range pyv.GetList(result, "items") {
			out = append(out, pyv.GetStr(item.(*pyjson.Object), "ref"))
		}
		return out
	}
	expect := func(name string, got []string, want ...string) {
		if len(got) != len(want) {
			t.Fatalf("%s: got %v, want %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: got %v, want %v", name, got, want)
			}
		}
	}
	classes := map[string]string{"IEDLD/XCBR1.Pos": "DPC", "IEDLD/MMTR1.SupWh": "BCR"}
	expect("DP", refs("point_kind=DP", nil), "IEDLD/XCBR1.Pos.stVal")
	expect("SP", refs("point_kind=SP", nil), "IEDLD/PTRC1.Tr.general")
	expect("CM without class", refs("point_kind=CM", nil), "IEDLD/XCBR1.Pos.Oper.ctlVal")
	expect("DC with class", refs("point_kind=DC", classes), "IEDLD/XCBR1.Pos.Oper.ctlVal")
	expect("CT with class", refs("point_kind=CT", classes), "IEDLD/MMTR1.SupWh.actVal")
	expect("IS without class", refs("point_kind=IS", nil), "IEDLD/MMTR1.SupWh.actVal")
	if got := refs("helpers=hide", nil); len(got) != 5 {
		t.Fatalf("helpers=hide kept %v", got)
	}
	values, _ := url.ParseQuery("kind_counts=1&point_kind=DP&helpers=hide")
	query := ParseQuery(values)
	query.Classes = classes
	result, err := ix.Search(query, nil)
	if err != nil {
		t.Fatal(err)
	}
	counts, _ := result.Get("kind_counts")
	body, _ := pyjson.Flask(counts)
	if strings.TrimSpace(string(body)) != `{"CT":1,"DC":1,"DP":1,"MV":1,"SP":1}` {
		t.Fatalf("kind_counts: %s", body)
	}
	if got := refs("", nil); len(got) != 6 {
		t.Fatalf("no filter: %v", got)
	}
}
