package modelcache

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"studio-engine/internal/pyjson"
)

func TestKeyFormula(t *testing.T) {
	source := "<SCL/>"
	sourceSum := sha256.Sum256([]byte(source))
	material := "v1\x00" + hex.EncodeToString(sourceSum[:]) + "\x00a.scd\x00IED1\x00IED2"
	sum := sha256.Sum256([]byte(material))
	if got := Key("v1", source, "a.scd", []string{"IED1", "IED2"}); got != hex.EncodeToString(sum[:]) {
		t.Fatalf("key %s", got)
	}
	if Key("v1", source, "a.scd", nil) == Key("v2", source, "a.scd", nil) {
		t.Fatal("engine version not in key")
	}
}

func TestPutGetStripsIdentityAndMissesDamage(t *testing.T) {
	cache := New(t.TempDir())
	key := Key("v1", "<SCL/>", "a.scd", nil)
	if _, ok := cache.Get(key); ok {
		t.Fatal("hit on empty cache")
	}
	ied := pyjson.NewObject().Set("id", "x").Set("name", "IED1").Set("source", "<SCL/>").Set("signals", []any{1.0})
	if err := cache.Put(key, []any{ied}); err != nil {
		t.Fatal(err)
	}
	if keys := ied.Keys(); len(keys) != 4 {
		t.Fatal("Put changed the caller's model")
	}
	models, ok := cache.Get(key)
	encoded, _ := pyjson.Dumps(models)
	if !ok || string(encoded) != `[{"name": "IED1", "signals": [1.0]}]` {
		t.Fatalf("got %s %v", encoded, ok)
	}
	stored, _ := os.ReadFile(cache.path(key))
	if string(stored) != string(encoded) {
		t.Fatalf("file %s", stored)
	}
	_ = os.WriteFile(cache.path(key), []byte("[1,"), 0o666)
	if _, ok := cache.Get(key); ok {
		t.Fatal("damaged entry was a hit")
	}
	if _, ok := cache.Get("../escape"); ok || cache.Put("../escape", nil) == nil {
		t.Fatal("invalid key accepted")
	}
}

func TestTrimKeepsNewestEntries(t *testing.T) {
	cache := New(t.TempDir())
	old := time.Now().Add(-time.Hour)
	var first string
	for index := 0; index < MaxEntries; index++ {
		key := Key("v1", fmt.Sprint(index), "a.scd", nil)
		if index == 0 {
			first = key
		}
		if err := cache.Put(key, []any{}); err != nil {
			t.Fatal(err)
		}
		at := old.Add(time.Duration(index) * time.Second)
		_ = os.Chtimes(cache.path(key), at, at)
	}
	cache.Get(first) // used again: becomes the newest
	for index := 0; index < 5; index++ {
		if err := cache.Put(Key("v1", fmt.Sprint("new", index), "a.scd", nil), []any{}); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := filepath.Glob(filepath.Join(cache.dir, "*.json"))
	if len(entries) != MaxEntries {
		t.Fatalf("%d entries", len(entries))
	}
	if _, ok := cache.Get(first); !ok {
		t.Fatal("recently used entry evicted")
	}
}
