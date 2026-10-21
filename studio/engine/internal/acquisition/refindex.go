package acquisition

import (
	"sync"

	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// refIndex keeps ref -> signal object maps per immutable IED model, built once per model
// object (Go rule "index once"); a new model object brings a new map.
type refIndex struct {
	mu    sync.Mutex
	byIED map[string]refEntry
}

type refEntry struct {
	ied  *pyjson.Object
	refs map[string]*pyjson.Object
}

// For returns the ref -> signal map of ied.
func (x *refIndex) For(ied *pyjson.Object) map[string]*pyjson.Object {
	id := pyv.GetStr(ied, "id")
	x.mu.Lock()
	entry, found := x.byIED[id]
	x.mu.Unlock()
	if found && entry.ied == ied {
		return entry.refs
	}
	refs := map[string]*pyjson.Object{}
	for _, signal := range pyv.Objects(pyv.GetList(ied, "signals")) {
		refs[pyv.GetStr(signal, "ref")] = signal
	}
	x.mu.Lock()
	if x.byIED == nil {
		x.byIED = map[string]refEntry{}
	}
	x.byIED[id] = refEntry{ied: ied, refs: refs}
	x.mu.Unlock()
	return refs
}

// findIED returns the IED with id from the published project, or nil.
func findIED(project *pyjson.Object, id string) *pyjson.Object {
	for _, ied := range pyv.Objects(pyv.GetList(project, "ieds")) {
		if pyv.GetStr(ied, "id") == id {
			return ied
		}
	}
	return nil
}
