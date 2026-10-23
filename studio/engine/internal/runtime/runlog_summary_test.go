package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/testutil"
)

// TestRunLogSummarisesGeneratorSamples checks the log volume rule (ADR-0115): generator
// samples become one signal_summary per IED and second (with the first/last bus event ids and
// each point's last value); every other event, including manual signal writes, stays one line.
func TestRunLogSummarisesGeneratorSamples(t *testing.T) {
	root := testutil.TempDir(t)
	log, err := OpenRunLog(root, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	id := int64(0)
	sample := func(ied, ref string, at float64, value float64) {
		id++
		log.Write(pyval.Obj("id", id, "kind", "signal", "timestamp", at, "ied_id", ied, "ref", ref,
			"value", value, "source", "generator", "quality", int64(0), "source_timestamp", int64(1)))
	}
	for i := 0; i < 20; i++ {
		sample("a", "LD/MMXU1.A.mag.f", 100.0+float64(i)*0.04, float64(i))
		sample("a", "LD/MMXU1.V.mag.f", 100.0+float64(i)*0.04, 2*float64(i))
	}
	sample("b", "LD/MMXU2.A.mag.f", 100.5, 7)
	id++
	log.Write(pyval.Obj("id", id, "kind", "signal", "timestamp", 100.9, "ied_id", "a", "ref", "LD/GGIO1.Ind1.stVal",
		"value", true, "source", "manual"))
	sample("a", "LD/MMXU1.A.mag.f", 101.2, 99)
	id++
	log.Write(pyval.Obj("id", id, "kind", "runtime", "timestamp", 101.5, "state", "stopping"))
	log.Close()
	events, _ := os.ReadFile(filepath.Join(root, log.RunID, "events-0000.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(events)), "\n")
	prefix := `{"run_id":"` + log.RunID + `",`
	want := []string{
		prefix + `"id":42,"kind":"signal","timestamp":100.9,"ied_id":"a","ref":"LD/GGIO1.Ind1.stVal","value":true,"source":"manual"}`,
		prefix + `"kind":"signal_summary","ied_id":"a","source":"generator","second":100,"samples":40,"from_id":1,"to_id":40,` +
			`"values":{"LD/MMXU1.A.mag.f":19.0,"LD/MMXU1.V.mag.f":38.0}}`,
		prefix + `"kind":"signal_summary","ied_id":"b","source":"generator","second":100,"samples":1,"from_id":41,"to_id":41,` +
			`"values":{"LD/MMXU2.A.mag.f":7.0}}`,
		prefix + `"id":44,"kind":"runtime","timestamp":101.5,"state":"stopping"}`,
		prefix + `"kind":"signal_summary","ied_id":"a","source":"generator","second":101,"samples":1,"from_id":43,"to_id":43,` +
			`"values":{"LD/MMXU1.A.mag.f":99.0}}`,
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Fatalf("events:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}
