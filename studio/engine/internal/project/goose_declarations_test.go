package project

import (
	"os"
	"testing"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// TestGooseDeclarationsGroupsAndCaches: stored files are grouped like stored_sources
// (one document per distinct text, IED names in project order) and the result is reused
// while the set of files is unchanged.
func TestGooseDeclarationsGroupsAndCaches(t *testing.T) {
	data, err := os.ReadFile("../scl/testdata/corpus/goose_sample_station.scd")
	if err != nil {
		t.Skip("corpus missing")
	}
	text := string(data)
	ied := func(name string) *pyjson.Object { return o("name", name, "filename", "s.scd", "source", text) }
	project := o("ieds", []any{ied("SUB1"), ied("PUB1")})
	var cache GooseDeclarations
	got, err := cache.For(project)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := scl.ReadGooseSources([]scl.GooseDocument{{Text: text, Loaded: []string{"SUB1", "PUB1"}}})
	a, _ := pyjson.Dumps(got)
	b, _ := pyjson.Dumps(want)
	if string(a) != string(b) {
		t.Fatal("declarations differ from read_goose_sources of the grouped files")
	}
	again, _ := cache.For(o("ieds", []any{ied("PUB1"), ied("SUB1")}))
	if again != got {
		t.Error("the same set of files should reuse the cached declarations")
	}
	empty, _ := cache.For(o("ieds", []any{}))
	if encoded, _ := pyjson.Dumps(empty); string(encoded) != `{"ieds": {}, "blocks": [], "subscriptions": {}, "problems": {}, "remote": {}}` {
		t.Errorf("empty declarations %s", encoded)
	}
	sources := StoredSources(project)
	if len(sources) != 1 || sources[0].ID != SourceDigest(text)[:16] || len(sources[0].IEDs) != 2 {
		t.Errorf("stored sources %+v", sources)
	}
}
