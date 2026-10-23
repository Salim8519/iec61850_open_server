package signalindex

import (
	"strconv"
	"testing"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// syntheticIED is a 90,000-signal model; testdata/gen has the same construction in Python
// for the timing comparison (refs LD<i%7>/GGIO<i/100>.Ind<i%100>.stVal).
func syntheticIED(count int) *pyjson.Object {
	signals := make([]any, count)
	for index := range signals {
		ref := "LD" + strconv.Itoa(index%7) + "/GGIO" + strconv.Itoa(index/100) + ".Ind" + strconv.Itoa(index%100)
		signals[index] = pyv.O("ref", ref+".stVal", "fc", "ST", "kind", "BOOLEAN", "value", false,
			"description", "Point "+strconv.Itoa(index), "enums", pyjson.NewObject(), "writable", true,
			"quality", int64(0), "timestamp", int64(0), "quality_ref", ref+".q", "timestamp_ref", ref+".t")
	}
	return pyv.O("id", "bench", "signals", signals, "datasets", []any{}, "reports", []any{})
}

func BenchmarkSearch(b *testing.B) {
	ix := Build(syntheticIED(90000))
	query := ParseQuery(ParseArgs("q=ind42&offset=10&limit=100"))
	b.ResetTimer()
	for range b.N {
		result, err := ix.Search(query, nil)
		if err != nil || pyv.Get(result, "total") != int64(900) {
			b.Fatalf("unexpected answer %v %v", pyv.Get(result, "total"), err)
		}
	}
}

func BenchmarkSearchAndEncode(b *testing.B) {
	ix := Build(syntheticIED(90000))
	query := ParseQuery(ParseArgs("q=ind42&offset=10&limit=100"))
	b.ResetTimer()
	for range b.N {
		result, _ := ix.Search(query, nil)
		if _, err := pyjson.Flask(result); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBuild(b *testing.B) {
	ied := syntheticIED(90000)
	b.ResetTimer()
	for range b.N {
		Build(ied)
	}
}

// syntheticClasses gives every data object of syntheticIED a class, as the SCL reader would.
func syntheticClasses(count int) map[string]string {
	classes := make(map[string]string, count)
	for index := range count {
		classes["LD"+strconv.Itoa(index%7)+"/GGIO"+strconv.Itoa(index/100)+".Ind"+strconv.Itoa(index%100)] = "SPS"
	}
	return classes
}

// BenchmarkKindFilter is the Go-only type filter with counts (ADR-0095) over the whole model.
func BenchmarkKindFilter(b *testing.B) {
	ix := Build(syntheticIED(90000))
	query := ParseQuery(ParseArgs("point_kind=SP&helpers=hide&kind_counts=1&limit=50"))
	query.Classes = syntheticClasses(90000)
	b.ResetTimer()
	for range b.N {
		result, err := ix.Search(query, nil)
		if err != nil || pyv.Get(result, "total") != int64(90000) {
			b.Fatalf("unexpected answer %v %v", pyv.Get(result, "total"), err)
		}
	}
}

// BenchmarkPickerQuery is one request of the smart point picker (ADR-0113) over the whole model:
// type filter, counts per type and per logical node, kinds on the items and suggestions.
func BenchmarkPickerQuery(b *testing.B) {
	ix := Build(syntheticIED(90000))
	query := ParseQuery(ParseArgs("point_kind=SP&helpers=hide&kind_counts=1&ln_counts=1&annotate=1&near=42&limit=50"))
	query.Classes = syntheticClasses(90000)
	b.ResetTimer()
	for range b.N {
		result, err := ix.Search(query, nil)
		if err != nil || pyv.Get(result, "total") != int64(90000) || len(pyv.GetList(result, "suggested")) != 1 {
			b.Fatalf("unexpected answer %v %v", pyv.Get(result, "total"), err)
		}
	}
}
