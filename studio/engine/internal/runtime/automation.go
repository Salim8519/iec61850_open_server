package runtime

import (
	"strings"
	"sync"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/runtime/signal"
)

// setSource stores a source; an existing key keeps its place (dict assignment). Needs c.mu.
func (c *Coordinator) setSource(key sourceKey, spec *source) {
	if _, ok := c.srcs[key]; !ok {
		c.srcOrder = append(c.srcOrder, key)
	}
	c.srcs[key] = spec
}

// dropSources forgets the sources of these IEDs (all when ids is nil). Needs c.mu.
func (c *Coordinator) dropSources(ids map[string]bool) {
	kept := c.srcOrder[:0]
	for _, key := range c.srcOrder {
		if ids == nil || ids[key.ied] {
			delete(c.srcs, key)
			continue
		}
		kept = append(kept, key)
	}
	c.srcOrder = kept
}

// loadSources copies the project's sources of these simulated IEDs (dict(item["spec"])).
func (c *Coordinator) loadSources(project *pyjson.Object, ids map[string]bool, start any) {
	for _, entry := range pyval.List(project, "sources") {
		object, _ := entry.(*pyjson.Object)
		if !ids[pyval.Str(object, "ied_id")] {
			continue
		}
		spec := pyval.Copy(pyval.Map(object, "spec")).(*pyjson.Object)
		if start != nil {
			spec.Set("start_elapsed", start)
		}
		overridden := pyval.Truthy(pyval.Get(spec, "overridden"))
		c.setSource(sourceKey{pyval.Str(object, "ied_id"), pyval.Str(object, "ref")}, &source{spec: spec, overridden: overridden})
	}
}

// automate is Automation.advance: evaluate every active source, then write each IED's batch
// of 80 points on its own actor, all IEDs in parallel, and wait for them.
func (c *Coordinator) automate(elapsed float64) {
	type planned struct {
		ied    string
		points []item
	}
	var groups []*planned
	byIed := map[string]*planned{}
	c.mu.Lock()
	keys := append([]sourceKey(nil), c.srcOrder...)
	specs := make([]*source, len(keys))
	for i, key := range keys {
		specs[i] = c.srcs[key]
	}
	c.mu.Unlock()
	for i, key := range keys {
		spec := specs[i]
		c.mu.Lock()
		skip := spec == nil || pyval.Get(spec.spec, "mode") == "manual" || spec.overridden
		c.mu.Unlock()
		if skip {
			continue
		}
		value, err := signal.ValueAt(spec.spec, elapsed)
		if err == nil {
			var info signal.Signal
			info, err = c.signalInfo(key.ied, key.ref)
			if err == nil && strings.HasPrefix(info.Kind, "INT") {
				value, err = pyval.ToInt(value)
			}
		}
		if err != nil {
			c.sourceFailed(key.ied, []item{{ref: key.ref}}, err)
			continue
		}
		group := byIed[key.ied]
		if group == nil {
			group = &planned{ied: key.ied}
			byIed[key.ied] = group
			groups = append(groups, group)
		}
		group.points = append(group.points, item{key.ref, value, nil})
	}
	var wait sync.WaitGroup
	for _, group := range groups {
		wait.Add(1)
		go func(group *planned) {
			defer wait.Done()
			for offset := 0; offset < len(group.points); offset += 80 {
				end := offset + 80
				if end > len(group.points) {
					end = len(group.points)
				}
				batch := group.points[offset:end]
				if c.State() != "running" {
					return
				}
				if _, err := c.apply(group.ied, batch, "generator"); err != nil {
					c.sourceFailed(group.ied, batch, err)
				}
			}
		}(group)
	}
	wait.Wait()
}

// sourceFailed is Automation.failed: pause the sources and say why.
func (c *Coordinator) sourceFailed(iedID string, points []item, err error) {
	for _, point := range points {
		c.mu.Lock()
		if spec := c.srcs[sourceKey{iedID, point.ref}]; spec != nil {
			spec.overridden = true
		}
		c.mu.Unlock()
		c.emit("source_failed", pyval.Obj("ied_id", iedID, "ref", point.ref, "message", errorText(err)))
	}
}
