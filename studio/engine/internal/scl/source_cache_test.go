package scl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Change 0185 item 12: a cached answer is found without hashing the source again, equal
// content in a new string reuses it, and the cache stays bounded.
func TestSourceCacheReusesAndBounds(t *testing.T) {
	var cache sourceCache
	builds := 0
	build := func() (any, error) { builds++; return map[string]string{"n": "x"}, nil }
	source := strings.Repeat("a", 1000)
	first, _ := cache.get("id", "IED", source, build)
	again, _ := cache.get("id", "IED", source, build)
	copied, _ := cache.get("id", "IED", strings.Clone(source), build)
	if builds != 1 || first.(map[string]string)["n"] != "x" || !sameMap(first, again) || !sameMap(first, copied) {
		t.Fatalf("builds %d, same %v %v", builds, sameMap(first, again), sameMap(first, copied))
	}
	for index := 0; index < 3*sourceCacheLimit; index++ {
		_, _ = cache.get("id", "IED", source+strings.Repeat("b", index+1), build)
	}
	if len(cache.identity) > sourceCacheLimit || len(cache.values) > sourceCacheLimit {
		t.Fatalf("cache grew to %d/%d entries", len(cache.identity), len(cache.values))
	}
}

func sameMap(a, b any) bool {
	left, right := a.(map[string]string), b.(map[string]string)
	left["probe"] = "1"
	defer delete(left, "probe")
	return right["probe"] == "1"
}

// BenchmarkCachedDataClasses measures one picker request's data-class lookup on a large
// SCL (set STUDIO_BENCH_SCL to a file; the bundled model substation otherwise).
func BenchmarkCachedDataClasses(b *testing.B) {
	path := os.Getenv("STUDIO_BENCH_SCL")
	if path == "" {
		path = filepath.Join(repoRoot, "scd", "model_substation.scd")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		b.Fatal(err)
	}
	source := string(data)
	name := os.Getenv("STUDIO_BENCH_IED")
	if name == "" {
		name = "FEED1"
	}
	if _, err := CachedDataClasses("bench", name, source); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = CachedDataClasses("bench", name, source)
	}
}
