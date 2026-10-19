// Package modelcache stores derived IED models (the SCL import result) outside the
// project, so reopening a project does not rebuild them (Go engine rule 8). The cache is
// untrusted: any damaged entry is a miss and the caller rebuilds from the SCL text.
//
// Format, for a Python `--derive-models` writer:
//
//	<data>/cache/models/<key>.json
//	key  = sha256 hex of UTF-8 bytes of
//	       engine + "\0" + sha256hex(source) + "\0" + filename + "\0" + "\0".join(selected)
//	       engine: the model builder version string; sha256hex(source): 64 lowercase hex of
//	       the decoded SCL text re-encoded as UTF-8 (the scl_sources digest); selected: the
//	       stored IED names in request order (import of all IEDs = empty list = "").
//	file = json.dumps(models) with default arguments (", " / ": ", ensure_ascii,
//	       insertion order, no newline), where models is import_scl's list with "id" and
//	       "source" removed from every IED dict (the caller re-attaches them).
//
// Writes are atomic (<key>.json.tmp-<random> then rename). After each Put at most the 64
// most recently used entries (by modification time; Get refreshes it) are kept.
package modelcache

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"studio-engine/internal/pyjson"
)

// Limits of the cache.
const (
	MaxEntries    = 64
	MaxEntryBytes = 256 << 20
)

// Cache is the folder <data>/cache/models.
type Cache struct {
	dir string
}

// New returns the cache of dataDir; the folder is created by the first Put.
func New(dataDir string) *Cache {
	return &Cache{dir: filepath.Join(dataDir, "cache", "models")}
}

// Key returns the entry key described in the package comment.
func Key(engine, source, filename string, selected []string) string {
	sourceSum := sha256.Sum256([]byte(source))
	material := engine + "\x00" + hex.EncodeToString(sourceSum[:]) + "\x00" + filename + "\x00" +
		strings.Join(selected, "\x00")
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

func validKey(key string) bool {
	if len(key) != 64 {
		return false
	}
	for _, c := range key {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func (c *Cache) path(key string) string { return filepath.Join(c.dir, key+".json") }

// Get returns the stored model list, or false on a miss (absent, damaged, invalid key).
func (c *Cache) Get(key string) ([]any, bool) {
	if !validKey(key) {
		return nil, false
	}
	path := c.path(key)
	info, err := os.Stat(path)
	if err != nil || info.Size() > MaxEntryBytes {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	value, err := pyjson.Decode(data)
	models, isList := value.([]any)
	if err != nil || !isList {
		return nil, false
	}
	now := time.Now()
	_ = os.Chtimes(path, now, now) // most recently used
	return models, true
}

// Put stores models (the import result; "id" and "source" are left out of every IED) and
// trims the cache to the newest MaxEntries entries.
func (c *Cache) Put(key string, models []any) error {
	if !validKey(key) {
		return os.ErrInvalid
	}
	stored := make([]any, len(models))
	for index, item := range models {
		stored[index] = withoutIdentity(item)
	}
	data, err := pyjson.Dumps(stored)
	if err != nil {
		return err
	}
	if len(data) > MaxEntryBytes {
		return nil // too large to be worth caching
	}
	if err := os.MkdirAll(c.dir, 0o777); err != nil {
		return err
	}
	var random [8]byte
	_, _ = rand.Read(random[:])
	temporary := c.path(key) + ".tmp-" + hex.EncodeToString(random[:])
	if err := os.WriteFile(temporary, data, 0o666); err != nil {
		return err
	}
	if err := os.Rename(temporary, c.path(key)); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	c.trim()
	return nil
}

func withoutIdentity(item any) any {
	ied, ok := item.(*pyjson.Object)
	if !ok {
		return item
	}
	copied := pyjson.NewObject()
	for _, key := range ied.Keys() {
		if key != "id" && key != "source" {
			value, _ := ied.Get(key)
			copied.Set(key, value)
		}
	}
	return copied
}

// trim removes the least recently used entries beyond MaxEntries and stale temp files.
func (c *Cache) trim() {
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		return
	}
	type entry struct {
		path string
		at   time.Time
	}
	var kept []entry
	for _, item := range entries {
		info, err := item.Info()
		if err != nil || item.IsDir() {
			continue
		}
		path := filepath.Join(c.dir, item.Name())
		if strings.Contains(item.Name(), ".tmp-") && time.Since(info.ModTime()) > time.Hour {
			_ = os.Remove(path)
		} else if strings.HasSuffix(item.Name(), ".json") {
			kept = append(kept, entry{path, info.ModTime()})
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].at.After(kept[j].at) })
	for index := MaxEntries; index < len(kept); index++ {
		_ = os.Remove(kept[index].path)
	}
}
