package acquisition

import (
	"sync"

	"studio-engine/internal/acquisition/modes"
	"studio-engine/internal/acquisition/pyv"
	"studio-engine/internal/pyjson"
)

// modeViews caches, per immutable IED model object, a view with every IED key except
// source and cfg, whose "signals" are only the Mod/Beh/Sim points. Health and mode reads
// use it instead of copying the model on every poll (Python change 0110 copied all keys).
type modeViews struct {
	mu    sync.Mutex
	byIED map[string]modeEntry
}

type modeEntry struct {
	ied  *pyjson.Object
	view *pyjson.Object
}

var modeCache = &modeViews{}

func (c *modeViews) view(ied *pyjson.Object) *pyjson.Object {
	id := pyv.GetStr(ied, "id")
	c.mu.Lock()
	entry, found := c.byIED[id]
	c.mu.Unlock()
	if found && entry.ied == ied {
		return entry.view
	}
	view := pyjson.NewObject()
	for _, key := range ied.Keys() {
		if key == "source" || key == "cfg" {
			continue
		}
		value, _ := ied.Get(key)
		if key == "signals" {
			value = modes.ModeSignals(pyv.List(value))
		}
		view.Set(key, value)
	}
	c.mu.Lock()
	if c.byIED == nil {
		c.byIED = map[string]modeEntry{}
	}
	c.byIED[id] = modeEntry{ied: ied, view: view}
	c.mu.Unlock()
	return view
}

// forget drops the views of IEDs that are no longer in the project.
func (c *modeViews) forget(keep map[string]bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.byIED {
		if !keep[id] {
			delete(c.byIED, id)
		}
	}
}
