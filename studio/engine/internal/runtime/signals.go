package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/runtime/signal"
)

// modelFor returns the run's model of an IED, or one built from the project when the IED is
// not in the run (Coordinator.signal falls back to the project's signal list).
func (c *Coordinator) modelFor(iedID string) (*iedModel, *valueTable, error) {
	c.mu.Lock()
	item := c.runs[iedID]
	stopped := c.state == "stopped"
	c.mu.Unlock()
	if item != nil && !stopped {
		return item.model, item.table, nil
	}
	raw := findIed(c.opt.Project.Current(), iedID)
	if raw == nil {
		return nil, nil, apperr.Coded("Unknown IED", "not_found")
	}
	model := newIedModel(raw)
	return model, c.tables.forModel(model), nil
}

// signalInfo is Coordinator.signal's lookup: the typed signal or Python's errors.
func (c *Coordinator) signalInfo(iedID, ref string) (signal.Signal, error) {
	model, _, err := c.modelFor(iedID)
	if err != nil {
		return signal.Signal{}, err
	}
	info, ok := model.lookup(ref)
	if !ok {
		return signal.Signal{}, apperr.ValueError("Unknown signal reference")
	}
	return info, nil
}

// Signal is Coordinator.signal: the signal dict with its runtime value, quality and time.
func (c *Coordinator) Signal(iedID, ref string) (*pyjson.Object, error) {
	model, table, err := c.modelFor(iedID)
	if err != nil {
		return nil, err
	}
	if _, ok := model.lookup(ref); !ok {
		return nil, apperr.ValueError("Unknown signal reference")
	}
	return table.view(ref), nil
}

// Value returns the runtime value of a signal (the "value" key Python kept on the signal
// dict), for the signals list and health views of other streams.
func (c *Coordinator) Value(iedID, ref string) (any, bool) {
	model, table, err := c.modelFor(iedID)
	if err != nil {
		return nil, false
	}
	if _, ok := model.lookup(ref); !ok {
		return nil, false
	}
	return table.get(ref)
}

// SourceState is what GET /project/test-panel shows for a signal: overridden and automatic.
func (c *Coordinator) SourceState(iedID, ref string) (overridden bool, automatic bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	spec := c.srcs[sourceKey{iedID, ref}]
	if spec == nil {
		return false, false
	}
	return spec.overridden, pyval.Get(spec.spec, "mode") != "manual"
}

// Update is POST /signals/update: one manual write; returns the value read back.
func (c *Coordinator) Update(iedID, ref string, value, quality any, source string) (any, error) {
	result, err := c.apply(iedID, []item{{ref, value, quality}}, source)
	if err != nil {
		return nil, err
	}
	actual, ok := result.Get(ref)
	if !ok {
		return nil, apperr.KeyError(ref)
	}
	return actual, nil
}

// ReleaseOverride is POST /signals/release-override.
func (c *Coordinator) ReleaseOverride(iedID, ref string) error {
	if !c.isLive() {
		return apperr.ValueError("Run the IED before resuming a source")
	}
	target := c.simulatedRun(iedID)
	if target == nil {
		return apperr.KeyError(iedID) // sample_writer.locks[ied_id] raised KeyError
	}
	return target.do(func() error {
		c.mu.Lock()
		spec := c.srcs[sourceKey{iedID, ref}]
		if spec == nil || pyval.Get(spec.spec, "mode") == "manual" {
			c.mu.Unlock()
			return apperr.ValueError("This signal has no automatic source")
		}
		c.mu.Unlock()
		c.pulses.cancel(iedID, ref, true, "Automatic source resumed")
		c.mu.Lock()
		spec.overridden = false
		c.mu.Unlock()
		c.emit("source_resumed", pyval.Obj("ied_id", iedID, "ref", ref))
		return nil
	})
}

// ConfigureSource is POST /signals/source: validate, store in the project, use at once.
func (c *Coordinator) ConfigureSource(iedID, ref string, spec any) error {
	info, err := c.signalInfo(iedID, ref)
	if err != nil {
		return err
	}
	candidate, err := signal.ValidateSource(info, spec)
	if err != nil {
		return err
	}
	if pyval.Get(candidate, "mode") != "manual" {
		value, err := signal.ValueAt(candidate, 0)
		if err != nil {
			return err
		}
		if len(info.Kind) < 5 || info.Kind[:5] != "FLOAT" {
			if value, err = pyval.ToInt(value); err != nil {
				return apperr.ValueError(err.Error())
			}
		}
		if _, err := signal.Validate(info.Kind, value); err != nil {
			return err
		}
	}
	if err := c.opt.Project.Source(iedID, ref, candidate); err != nil {
		return err
	}
	stored := pyval.Copy(candidate).(*pyjson.Object)
	c.mu.Lock()
	if c.live() {
		stored.Set("start_elapsed", c.elapsedValue())
	} else {
		stored.Set("start_elapsed", int64(0))
	}
	c.setSource(sourceKey{iedID, ref}, &source{spec: stored})
	c.mu.Unlock()
	c.emit("source", pyval.Obj("ied_id", iedID, "ref", ref, "spec", signal.Summary(stored)))
	return nil
}
