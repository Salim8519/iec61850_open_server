package svdom

import (
	"strings"
	"testing"

	"studio-engine/internal/acquisition/goldentest"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

func objectList(items []*pyjson.Object) []any {
	result := make([]any, len(items))
	for index, item := range items {
		result[index] = item
	}
	return result
}

func check(c goldentest.Case) any {
	function, _, _ := strings.Cut(c.Name, "/")
	switch function {
	case "expected_rate":
		return ExpectedRate(c.Obj(0), c.Args[1])
	case "samples_per_cycle":
		return SamplesPerCycle(c.Obj(0), c.Args[1])
	case "watch_list":
		supervised, rest := WatchList(pyv.Objects(pyv.List(c.Args[0])), c.Args[1])
		return []any{objectList(supervised), objectList(rest)}
	case "native_stream":
		return NativeStream(c.Obj(0))
	case "assess":
		return Assess(c.Obj(0), c.Obj(1), c.Obj(2))
	}
	return "no Go check for " + function
}

// TestSupervisionMatchesPython compares json.dumps bytes (insertion order kept, 4000 vs
// 4000.0 distinguished) with the Python recording.
func TestSupervisionMatchesPython(t *testing.T) {
	seen := map[string]int{}
	for _, c := range goldentest.Load(t, "sv_supervision.json") {
		body, err := pyjson.Dumps(check(c))
		if err != nil {
			t.Fatalf("%s: encode: %v", c.Name, err)
		}
		goldentest.Text(t, c, string(body))
		function, _, _ := strings.Cut(c.Name, "/")
		seen[function]++
	}
	for _, function := range []string{"expected_rate", "samples_per_cycle", "watch_list", "native_stream", "assess"} {
		if seen[function] == 0 {
			t.Errorf("no golden cases for %s", function)
		}
	}
}

func TestProblemOrderAndLimits(t *testing.T) {
	order := ProblemOrder()
	if len(order) != 7 || order[0] != "conf_rev_mismatch" || order[6] != "simulation" {
		t.Fatalf("ProblemOrder = %v", order)
	}
	if StreamLimit != 16 || RateTolerance != 0.02 {
		t.Fatal("constants differ from sv_supervision.py")
	}
}
