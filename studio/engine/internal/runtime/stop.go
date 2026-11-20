package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/memtrim"
	"studio-engine/internal/runtime/pyval"
)

// Stop is POST /runtime/stop (ids nil) and /runtime/ieds/<id>/stop: stop the whole run, or
// only these IEDs while the others keep running (Coordinator.stop).
func (c *Coordinator) Stop(ids []string) error {
	defer memtrim.Hold()()
	c.lifeMu.Lock()
	defer c.lifeMu.Unlock()
	project := c.opt.Project.Current()
	for _, id := range ids {
		if findIed(project, id) == nil { // not "This IED is not running" (change 0186)
			return apperr.Coded("Unknown IED", "not_found")
		}
	}
	c.mu.Lock()
	live := c.live()
	partial := false
	if ids != nil && live {
		selection := selectionOf(ids)
		for id := range c.active {
			partial = partial || !selection[id]
		}
	}
	c.mu.Unlock()
	if partial {
		return c.removeIeds(selectionOf(ids))
	}
	c.pulses.cancel("", "", false, "Simulation stopped")
	c.mu.Lock()
	c.setStateLocked("stopping")
	items := make([]*running, 0, len(c.runs))
	for _, id := range sortedKeys(c.runs) {
		items = append(items, c.runs[id])
	}
	c.mu.Unlock()
	// Real-IED sessions give back their report blocks while the workers close (change 0223); the
	// IEDs are marked stopped after both, so the events keep their order.
	sessions := make(chan struct{})
	go func() {
		defer close(sessions)
		c.opt.Live.Stop()
	}()
	c.feedback.clear()
	c.selections.clear()
	c.scenarios.stop()
	simulated := simulatedOf(items)
	c.closeWorkers(simulated)
	<-sessions
	c.markWorkersStopped(simulated)
	c.markAllStopped(c.opt.Project.Current())
	c.mu.Lock()
	c.setStateLocked("stopped")
	c.active, c.runs = map[string]bool{}, map[string]*running{}
	c.mu.Unlock()
	c.opt.LabRelease()
	c.emit("runtime", pyval.Obj("state", "stopped"))
	if log := c.setLog(nil); log != nil {
		log.Close()
	}
	c.PrepareSpares()
	return nil
}

// Pause is POST /runtime/pause: toggles automation between running and paused.
func (c *Coordinator) Pause() error {
	c.lifeMu.Lock()
	defer c.lifeMu.Unlock()
	c.mu.Lock()
	if !c.live() {
		c.mu.Unlock()
		return apperr.ValueError("Start the simulation before pausing automation")
	}
	now := c.now()
	if c.state == "running" {
		c.addElapsed(now - c.lastAutomation)
	}
	c.lastAutomation = now
	c.cadence.reset(now)
	if c.state == "running" {
		c.setStateLocked("paused")
	} else {
		c.setStateLocked("running")
	}
	state := c.state
	c.mu.Unlock()
	c.emit("runtime", pyval.Obj("state", state))
	return nil
}
