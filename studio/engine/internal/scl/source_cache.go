package scl

import (
	"hash/fnv"
	"strconv"
	"sync"
	"unsafe"
)

// sourceCacheLimit bounds a cache: a project holds at most 20 IEDs, so 64 entries keep every
// IED of the open project and a few earlier ones; above it the cache starts again.
const sourceCacheLimit = 64

// identity is one IED's source text by identity (address and length), the way
// project.SourceID keys its digests: the entry holds the text, so the address is not reused.
type identity struct {
	id, name string
	data     *byte
	size     int
}

type identityEntry struct {
	text string
	key  string // "<id>|<name>|<content hash>"
}

// sourceCache remembers a value per IED and source text. A request with the same text value
// (the project passes the same string each time) costs one map lookup; a new string is
// hashed once and reuses the value of equal content (ADR-0114, change 0185).
type sourceCache struct {
	mu       sync.Mutex
	identity map[identity]identityEntry
	values   map[string]any
}

func (c *sourceCache) get(id, name, source string, build func() (any, error)) (any, error) {
	key := identity{id, name, unsafe.StringData(source), len(source)}
	c.mu.Lock()
	if entry, ok := c.identity[key]; ok {
		if value, found := c.values[entry.key]; found {
			c.mu.Unlock()
			return value, nil
		}
	}
	c.mu.Unlock()
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(source))
	contentKey := id + "|" + name + "|" + strconv.FormatUint(hash.Sum64(), 16)
	c.mu.Lock()
	value, found := c.values[contentKey]
	c.mu.Unlock()
	if !found {
		built, err := build()
		if err != nil {
			return nil, err
		}
		value = built
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.identity == nil || len(c.identity) >= sourceCacheLimit || len(c.values) >= sourceCacheLimit {
		c.identity, c.values = map[identity]identityEntry{}, map[string]any{}
	}
	if existing, ok := c.values[contentKey]; ok {
		value = existing // a concurrent request stored it first: keep one value per content
	}
	c.values[contentKey] = value
	c.identity[key] = identityEntry{text: source, key: contentKey}
	return value, nil
}
