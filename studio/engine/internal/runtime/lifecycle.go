package runtime

import (
	"fmt"
	"path/filepath"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// runnable is run_selection.runnable: IEDs not set to Off, or only the selected ones.
func runnable(project *pyjson.Object, selection map[string]bool, order []string) ([]*pyjson.Object, error) {
	var ieds []*pyjson.Object
	for _, ied := range projectIeds(project) {
		if roleOf(ied) != "off" {
			ieds = append(ieds, ied)
		}
	}
	if selection == nil {
		return ieds, nil
	}
	for _, id := range order {
		ied := findIed(project, id)
		if ied == nil {
			return nil, apperr.ValueError("Unknown IED")
		}
		if roleOf(ied) == "off" {
			return nil, apperr.ValueError(fmt.Sprintf("%s is left out. Choose 'Be this IED' or 'Read real IED' first.",
				pyval.Str(ied, "name")))
		}
	}
	var chosen []*pyjson.Object
	for _, ied := range ieds {
		if selection[pyval.Str(ied, "id")] {
			chosen = append(chosen, ied)
		}
	}
	return chosen, nil
}

// copyIed is a shallow copy so a run can store its assigned address without touching the
// immutable project tree.
func copyIed(ied *pyjson.Object) *pyjson.Object {
	copied := pyjson.NewObject()
	for _, key := range ied.Keys() {
		value, _ := ied.Get(key)
		copied.Set(key, value)
	}
	return copied
}

func (c *Coordinator) newRuns(ieds []*pyjson.Object) []*running {
	items := make([]*running, len(ieds))
	for i, ied := range ieds {
		model := newIedModel(ied)
		items[i] = newRunning(model, c.tables.forModel(model), copyIed(ied))
	}
	return items
}

func simulatedOf(items []*running) []*running {
	var simulated []*running
	for _, item := range items {
		if item.simulated() {
			simulated = append(simulated, item)
		}
	}
	return simulated
}

// Start is POST /runtime/start (ids nil) and /runtime/ieds/<id>/start (Coordinator.start).
func (c *Coordinator) Start(ids []string) error {
	c.lifeMu.Lock()
	defer c.lifeMu.Unlock()
	c.mu.Lock()
	live := c.live()
	c.mu.Unlock()
	if ids != nil && live {
		return c.addIeds(ids)
	}
	if live {
		project := c.opt.Project.Current()
		all, _ := runnable(project, nil, nil)
		var rest []string
		c.mu.Lock()
		for _, ied := range all {
			if id := pyval.Str(ied, "id"); !c.active[id] {
				rest = append(rest, id)
			}
		}
		c.mu.Unlock()
		if len(rest) > 0 {
			return c.addIeds(rest)
		}
	}
	return c.start(ids)
}

func selectionOf(ids []string) map[string]bool {
	if ids == nil {
		return nil
	}
	selection := map[string]bool{}
	for _, id := range ids {
		selection[id] = true
	}
	return selection
}

func (c *Coordinator) start(ids []string) error {
	project := c.opt.Project.Current()
	c.mu.Lock()
	if c.state != "stopped" {
		c.mu.Unlock()
		return apperr.ValueError("Stop the current simulation first")
	}
	c.mu.Unlock()
	chosen, err := runnable(project, selectionOf(ids), ids)
	if err != nil {
		return err
	}
	if len(projectIeds(project)) == 0 {
		return apperr.ValueError("Import or create an IED before Run")
	}
	if len(chosen) == 0 {
		return apperr.ValueError("Set at least one IED to 'Be this IED' or 'Read real IED' before Run")
	}
	network := pyval.Map(project, "network")
	if conflicts := endpointConflicts(chosen, network); len(conflicts) > 0 {
		return apperr.ValueError(conflicts[0])
	}
	items := c.newRuns(chosen)
	simulated := simulatedOf(items)
	diagram := pyval.Map(project, "diagram")
	c.mu.Lock()
	c.setStateLocked("starting")
	c.active, c.runs = map[string]bool{}, map[string]*running{}
	c.reindexLocked(items)
	simIDs := map[string]bool{}
	for _, item := range items {
		c.active[item.model.id] = true
		c.runs[item.model.id] = item
		if item.simulated() {
			simIDs[item.model.id] = true
		}
	}
	c.srcs, c.srcOrder = map[sourceKey]*source{}, nil
	c.loadSources(project, simIDs, nil)
	c.elapsed, c.elapsedFloat = 0, false
	c.mu.Unlock()
	c.access.reset()
	c.plant.reset(diagram)
	c.feedback.reset(diagram)
	if err := c.startRun(project, items, simulated); err != nil {
		c.opt.Live.Stop()
		c.feedback.clear()
		c.scenarios.stop()
		c.stopWorkers(simulated)
		c.markAllStopped(project)
		c.mu.Lock()
		c.setStateLocked("stopped")
		c.active, c.runs = map[string]bool{}, map[string]*running{}
		c.mu.Unlock()
		c.emit("runtime", pyval.Obj("state", "stopped", "message", errorText(err)))
		if log := c.setLog(nil); log != nil {
			log.Close()
		}
		return err
	}
	return nil
}

func (c *Coordinator) startRun(project *pyjson.Object, items, simulated []*running) error {
	log, err := OpenRunLog(filepath.Join(c.opt.DataDir, "logs"))
	if err != nil {
		return apperr.RuntimeError(err.Error())
	}
	c.setLog(log)
	if len(simulated) > 0 {
		if err := c.addWorkers(simulated, pyval.Map(project, "network"), nil); err != nil {
			return err
		}
	}
	c.scenarios.reset(pyval.List(project, "scenarios"))
	c.mu.Lock()
	c.setStateLocked("running")
	c.lastAutomation = c.now()
	active := c.sortedActive()
	c.mu.Unlock()
	c.emit("runtime", pyval.Obj("state", "running", "running", stringsAny(active)))
	if err := c.publishBindings(project, nil); err != nil {
		return err
	}
	if err := c.applyInitialStates(project, nil); err != nil {
		return err
	}
	if err := c.applyTestModes(project, nil); err != nil {
		return err
	}
	if err := c.opt.Live.Start(project, active); err != nil {
		return err
	}
	c.mu.Lock()
	c.lastAutomation = c.now()
	c.cadence.reset(c.lastAutomation)
	c.mu.Unlock()
	return nil
}

func stringsAny(items []string) []any {
	values := make([]any, len(items))
	for i, item := range items {
		values[i] = item
	}
	return values
}

// markAllStopped is Supervisor.stop's last loop: every project IED is "stopped".
func (c *Coordinator) markAllStopped(project *pyjson.Object) {
	for _, ied := range projectIeds(project) {
		c.markStopped(pyval.Str(ied, "id"))
	}
}
