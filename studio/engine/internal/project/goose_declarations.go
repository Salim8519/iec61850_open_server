package project

import (
	"sort"
	"strings"
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/scl"
)

// GooseDeclarations is projects/goose_links._declarations: the merged GOOSE declarations
// of the project's stored SCL files (scl.ReadGooseSources, JSON layout), kept for the
// last set of files because parsing vendor files takes seconds. The returned object is
// shared: never modify it. The zero value is ready to use.
//
// The live area wants goosedom.Declarations: the composition passes
// goosedom.DeclarationsFromJSON(value) (projects must not import acquisition).
type GooseDeclarations struct {
	mu    sync.Mutex
	key   string
	value *pyjson.Object
}

// For returns the declarations of project's stored files (empty ones without files).
func (c *GooseDeclarations) For(project *pyjson.Object) (*pyjson.Object, error) {
	sources := StoredSources(project)
	if len(sources) == 0 {
		return emptyDeclarations(), nil
	}
	key := declarationKey(sources)
	c.mu.Lock()
	cached := c.value
	if c.key != key {
		cached = nil
	}
	c.mu.Unlock()
	if cached != nil {
		return cached, nil
	}
	documents := make([]scl.GooseDocument, len(sources))
	for index, entry := range sources {
		documents[index] = scl.GooseDocument{Text: entry.Text, Loaded: entry.IEDs}
	}
	value, err := scl.ReadGooseSources(documents)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.key, c.value = key, value
	c.mu.Unlock()
	return value, nil
}

// declarationKey is tuple(sorted((entry["id"], tuple(sorted(entry["ieds"]))) ...)).
func declarationKey(sources []StoredSource) string {
	parts := make([]string, len(sources))
	for index, entry := range sources {
		names := append([]string(nil), entry.IEDs...)
		sort.Strings(names)
		parts[index] = entry.ID + "\x00" + strings.Join(names, "\x00")
	}
	sort.Strings(parts)
	return strings.Join(parts, "\x01")
}

func emptyDeclarations() *pyjson.Object {
	return pyjson.NewObject().Set("ieds", pyjson.NewObject()).Set("blocks", []any{}).
		Set("subscriptions", pyjson.NewObject()).Set("problems", pyjson.NewObject()).Set("remote", pyjson.NewObject())
}
