package xsd

import (
	"os"
	"path/filepath"
	"reflect"
	"studio-engine/internal/testutil"
	"testing"

	"studio-engine/internal/scl"
)

var sampleVerdict = scl.SchemaVerdict{Status: "invalid", Errors: []scl.SchemaError{
	{Line: 2, Message: "Element 'é': This element is not expected."},
	{Line: 70000, Message: "second \"quoted\" message"},
}}

func TestCacheFileIsPythonJSON(t *testing.T) {
	data, err := encodeVerdict(sampleVerdict)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"status": "invalid", "errors": [{"line": 2, "message": "Element '` + "\\" + `u00e9': This element is not expected."}, ` +
		`{"line": 70000, "message": "second \"quoted\" message"}]}`
	if string(data) != want {
		t.Fatalf("cache file bytes\n got %s\nwant %s", data, want)
	}
	back, err := decodeVerdict(data)
	if err != nil || !reflect.DeepEqual(back, sampleVerdict) {
		t.Fatalf("decode: %+v %v", back, err)
	}
}

func TestCacheHitSurvivesRestart(t *testing.T) {
	dir := testutil.TempDir(t)
	key := cacheKey(sha256Hex([]byte("<SCL/>")), "SCL2007B", "digest")
	if err := newCache(dir, 4).put(key, sampleVerdict); err != nil {
		t.Fatal(err)
	}
	verdict, found := newCache(dir, 4).get(key) // a new process: disk only
	if !found || !reflect.DeepEqual(verdict, sampleVerdict) {
		t.Fatalf("disk hit: %v %+v", found, verdict)
	}
	verdict.Errors[0].Message = "changed by the caller"
	again, _ := newCache(dir, 4).get(key)
	if again.Errors[0].Message != sampleVerdict.Errors[0].Message {
		t.Fatal("callers must not be able to change cached verdicts")
	}
	if _, found := newCache(dir, 4).get(key + "x"); found {
		t.Fatal("another key must miss")
	}
}

func TestCacheCorruptFileIsMiss(t *testing.T) {
	dir := testutil.TempDir(t)
	key := cacheKey(sha256Hex([]byte("doc")), "SCL2003", "digest")
	store := newCache(dir, 4)
	for name, content := range map[string]string{
		"truncated":   `{"status": "valid", "errors": [`,
		"bad status":  `{"status": "unsupported", "errors": []}`,
		"extra key":   `{"status": "valid", "errors": [], "schema": "SCL2003"}`,
		"bad line":    `{"status": "invalid", "errors": [{"line": "2", "message": "x"}]}`,
		"empty":       ``,
		"not object":  `[]`,
		"float line":  `{"status": "invalid", "errors": [{"line": 2.0, "message": "x"}]}`,
		"missing msg": `{"status": "invalid", "errors": [{"line": 2}]}`,
	} {
		if err := os.WriteFile(store.file(key), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if verdict, found := store.get(key); found {
			t.Errorf("%s: corrupt file must be a miss, got %+v", name, verdict)
		}
	}
}

func TestCacheKeyCoversLibxml2AndDigest(t *testing.T) {
	key := cacheKey("aa", "SCL2007B4", "dd")
	if key != "aa\x00SCL2007B4\x002.11.9\x00dd" {
		t.Fatalf("unexpected key %q", key)
	}
	name := filepath.Base(newCache("x", 1).file(key))
	if len(name) != 64+len(".json") {
		t.Fatalf("cache file name %q is not sha256(key).json", name)
	}
}

func TestMemoryLRUEvictsOldest(t *testing.T) {
	memory := newLRU(2)
	memory.put("a", sampleVerdict)
	memory.put("b", sampleVerdict)
	memory.get("a")
	memory.put("c", sampleVerdict)
	if _, found := memory.get("b"); found {
		t.Fatal("least recently used entry must be evicted")
	}
	if _, found := memory.get("a"); !found || memory.len() != 2 {
		t.Fatal("recently used entry must stay")
	}
}

func TestSchemaDigestReadsOnlyXSDFiles(t *testing.T) {
	digests := newSchemaDigests(bundledSchemas)
	first, err := digests.get("SCL2007B4")
	if err != nil || len(first) != 64 {
		t.Fatalf("digest: %q %v", first, err)
	}
	other, _ := digests.get("SCL2003")
	if other == first {
		t.Fatal("different schema folders must have different digests")
	}
	if _, err := newSchemaDigests(testutil.TempDir(t)).get("SCL2007B"); err == nil {
		t.Fatal("a missing schema folder must fail")
	} else {
		requireRuntime(t, err, "bundled XSD files for SCL2007B")
	}
}
