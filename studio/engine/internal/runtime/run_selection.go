package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// addIeds is run_selection.add_ieds: start more IEDs inside the current run.
func (c *Coordinator) addIeds(ids []string) error {
	project := c.opt.Project.Current()
	chosen, err := runnable(project, selectionOf(ids), ids)
	if err != nil {
		return err
	}
	c.mu.Lock()
	var fresh []*pyjson.Object
	for _, ied := range chosen {
		if !c.active[pyval.Str(ied, "id")] {
			fresh = append(fresh, ied)
		}
	}
	var current []*running
	for _, ied := range projectIeds(project) {
		if item := c.runs[pyval.Str(ied, "id")]; item != nil && c.active[item.model.id] {
			current = append(current, item)
		}
	}
	c.mu.Unlock()
	if len(fresh) == 0 {
		return apperr.ValueError("This IED is already running")
	}
	everyone := make([]*pyjson.Object, 0, len(current)+len(fresh))
	for _, item := range current {
		everyone = append(everyone, item.ied)
	}
	everyone = append(everyone, fresh...)
	if conflicts := endpointConflicts(everyone, pyval.Map(project, "network")); len(conflicts) > 0 {
		return apperr.ValueError(conflicts[0])
	}
	items := c.newRuns(fresh)
	simulated := simulatedOf(items)
	ids2 := map[string]bool{}
	simIDs := map[string]bool{}
	var real []*pyjson.Object
	c.mu.Lock()
	for _, item := range items {
		ids2[item.model.id] = true
		c.runs[item.model.id] = item
		c.active[item.model.id] = true
		c.indexLocked(item)
		if item.simulated() {
			simIDs[item.model.id] = true
		} else if item.model.role == "real" {
			real = append(real, item.ied)
		}
	}
	c.loadSources(project, simIDs, c.elapsedValue())
	c.mu.Unlock()
	if len(simulated) > 0 {
		if err := c.addWorkers(simulated, pyval.Map(project, "network"), simulatedOf(current)); err != nil {
			c.forget(items)
			return err
		}
	}
	if err := c.opt.Live.Connect(project, real); err != nil {
		c.forget(items)
		return err
	}
	if err := c.publishBindings(project, ids2); err != nil {
		return err
	}
	if err := c.applyInitialStates(project, ids2); err != nil {
		return err
	}
	if err := c.applyTestModes(project, ids2); err != nil {
		return err
	}
	c.plant.publishAll()
	for _, item := range items {
		c.emit("ied_runtime", pyval.Obj("ied_id", item.model.id, "state", "started", "role", item.model.role))
	}
	c.emitRuntime()
	return nil
}

func (c *Coordinator) emitRuntime() {
	c.mu.Lock()
	state, active := c.state, c.sortedActive()
	c.mu.Unlock()
	c.emit("runtime", pyval.Obj("state", state, "running", stringsAny(active)))
}

// removeIeds is run_selection.remove_ieds: stop some IEDs; the others keep running.
func (c *Coordinator) removeIeds(selection map[string]bool) error {
	var gone []*running
	c.mu.Lock()
	for _, ied := range projectIeds(c.opt.Project.Current()) {
		id := pyval.Str(ied, "id")
		if selection[id] && c.active[id] && c.runs[id] != nil {
			gone = append(gone, c.runs[id])
		}
	}
	c.mu.Unlock()
	if len(gone) == 0 {
		return apperr.ValueError("This IED is not running")
	}
	for _, item := range gone {
		c.pulses.cancel(item.model.id, "", false, "IED stopped")
	}
	c.forget(gone)
	c.plant.publishAll()
	for _, item := range gone {
		c.emit("ied_runtime", pyval.Obj("ied_id", item.model.id, "state", "stopped", "role", item.model.role))
	}
	c.emitRuntime()
	return nil
}

// forget is run_selection._forget.
func (c *Coordinator) forget(items []*running) {
	c.stopWorkers(simulatedOf(items))
	ids := map[string]bool{}
	var list []string
	for _, item := range items {
		ids[item.model.id] = true
		list = append(list, item.model.id)
		if item.model.role == "real" {
			c.opt.Live.Disconnect(item.model.id)
		}
	}
	c.mu.Lock()
	for id := range ids {
		delete(c.active, id)
		delete(c.runs, id)
		c.unindexLocked(id)
	}
	c.dropSources(ids)
	c.mu.Unlock()
	c.access.forget(list)
}
