package service

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
	"studio-engine/internal/store/modelcache"
)

// storedModels re-imports one stored SCL source for restore. With a model cache the
// derived models of an unchanged source come from <data>/cache/models instead of a new
// parse, type resolution and XSD check; ids are still drawn in import order and the
// source text is re-attached where import_scl puts it, so the result is identical.
func (s *Service) storedModels(source string, filename any, names []any) ([]*pyjson.Object, error) {
	cache := s.config.ModelCache
	key, cacheable := "", false
	if cache != nil {
		key, cacheable = modelKey(source, filename, names)
	}
	if cacheable {
		if cached, ok := cache.Get(key); ok {
			if models, ok := s.attachIdentity(cached, source); ok {
				return models, nil
			}
		}
	}
	built, err := s.config.Importer.ImportNamed([]byte(source), filename, scl.Selection{Names: names})
	if err != nil {
		return nil, err
	}
	if cacheable {
		_ = cache.Put(key, objects(built)) // best effort: a failed write is only a later miss
	}
	return built, nil
}

// modelKey is the cache key, available for text filenames and names only.
func modelKey(source string, filename any, names []any) (string, bool) {
	file, ok := filename.(string)
	if !ok {
		return "", false
	}
	selected := make([]string, len(names))
	for index, name := range names {
		text, ok := name.(string)
		if !ok {
			return "", false
		}
		selected[index] = text
	}
	return modelcache.Key(scl.ModelVersion, source, file, selected), true
}

// attachIdentity rebuilds each model as import_scl orders it: id, name, filename, source,
// then the cached fields. A malformed entry is a miss (no id is drawn).
func (s *Service) attachIdentity(cached []any, source string) ([]*pyjson.Object, bool) {
	for _, item := range cached {
		model, ok := item.(*pyjson.Object)
		if !ok || model.Len() < 2 || model.Keys()[0] != "name" || model.Keys()[1] != "filename" {
			return nil, false
		}
	}
	text := strings.TrimPrefix(source, string(rune(0xfeff))) // source.decode("utf-8-sig")
	newID := s.config.Importer.NewID
	if newID == nil {
		newID = scl.RandomID
	}
	models := make([]*pyjson.Object, len(cached))
	for index, item := range cached {
		model := item.(*pyjson.Object)
		rebuilt := pyjson.NewObject().Set("id", newID())
		for position, key := range model.Keys() {
			value, _ := model.Get(key)
			rebuilt.Set(key, value)
			if position == 1 {
				rebuilt.Set("source", text)
			}
		}
		models[index] = rebuilt
	}
	return models, true
}
