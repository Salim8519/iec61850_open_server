package service

import (
	"runtime"
	"sync"

	"studio-engine/internal/project"
)

// cachedRead is one source's model-cache entry, read before its turn in a restore.
type cachedRead struct {
	models []any
	hit    bool
}

// readCached reads and decodes the cached models of every stored source at once, at most one
// per processor (ADR-0133: opening a 7-IED project decoded ~90 MB of cached JSON one file
// after another before the page could load). Only the reading happens early: ids, imports and
// errors still follow the stored order in Restore. A source that is not cacheable gets nil.
func (s *Service) readCached(groups []*project.StoredGroup) []*cachedRead {
	reads := make([]*cachedRead, len(groups))
	cache := s.config.ModelCache
	if cache == nil {
		return reads
	}
	slots := make(chan struct{}, max(1, min(runtime.NumCPU(), len(groups))))
	var group sync.WaitGroup
	for index, stored := range groups {
		names, err := stored.Names()
		source, isText := stored.Source.(string)
		if err != nil || !isText {
			continue // Restore reports it in order
		}
		key, cacheable := modelKey(source, stored.Filename, names)
		if !cacheable {
			continue
		}
		group.Add(1)
		go func(index int, key string) {
			defer group.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			read := &cachedRead{}
			read.models, read.hit = cache.Get(key)
			reads[index] = read
		}(index, key)
	}
	group.Wait()
	return reads
}
