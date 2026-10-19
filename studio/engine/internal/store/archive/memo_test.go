package archive

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"testing"

	"studio-engine/internal/pyjson"
)

// memoDoc is a document with shared subtrees, every scalar kind and nesting deeper than
// the memoised levels.
func memoDoc(nodes int, shared *pyjson.Object) *pyjson.Object {
	list := []any{}
	for n := 0; n < nodes; n++ {
		list = append(list, pyjson.NewObject().Set("id", fmt.Sprintf("n%dé\"", n)).
			Set("data", pyjson.NewObject().Set("deep", []any{int64(n), 1.5, true, nil, big.NewInt(1 << 62), "x\n"})))
	}
	list = append(list, shared, shared)
	return pyjson.NewObject().Set("diagram", pyjson.NewObject().Set("nodes", list).Set("edges", []any{})).
		Set("empty", pyjson.NewObject()).Set("n", int64(3)).Set("f", 0.1)
}

// TestMemoDumpsMatchesPyjson: memoised encoding is byte-identical to pyjson.Dumps across
// edits that share and replace subtrees, and refuses non-finite floats like json.dumps.
func TestMemoDumpsMatchesPyjson(t *testing.T) {
	encoder := NewEncoder()
	shared := pyjson.NewObject().Set("k", []any{"v"})
	doc := memoDoc(50, shared)
	for round := 0; round < 30; round++ {
		encoder.nextGeneration()
		got, err := encoder.Dumps(doc)
		want, _ := pyjson.Dumps(doc)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("round %d: memo encoding differs (%v)", round, err)
		}
		next := memoDoc(0, shared)
		diagram, _ := doc.Get("diagram")
		nodes, _ := diagram.(*pyjson.Object).Get("nodes")
		edited := append([]any(nil), nodes.([]any)...)
		edited[round%len(edited)] = pyjson.NewObject().Set("id", fmt.Sprintf("edit %d", round))
		next.Set("diagram", pyjson.NewObject().Set("nodes", edited).Set("edges", []any{}))
		doc = next
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		value := pyjson.NewObject().Set("a", []any{pyjson.NewObject().Set("b", []any{bad})})
		_, memoErr := encoder.Dumps(value)
		plainErr := checkFinite(value)
		if memoErr == nil || plainErr == nil || memoErr.Error() != plainErr.Error() {
			t.Errorf("non-finite %v: memo %v, plain %v", bad, memoErr, plainErr)
		}
	}
}
