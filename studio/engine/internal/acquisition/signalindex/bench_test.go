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
