package runtime

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/plant"
	"studio-engine/internal/runtime/pyval"
)

// publishBindings is simulation/bindings.publish_bindings: read every mapped canvas and
// test-panel point from the workers once, emit it, then publish all equipment states.
func (c *Coordinator) publishBindings(project *pyjson.Object, only map[string]bool) error {
	var bindings []plant.Binding
	seen := map[plant.Binding]bool{}
	add := func(binding plant.Binding) {
		if !seen[binding] {
			seen[binding] = true
			bindings = append(bindings, binding)
		}
	}
	for _, node := range diagramNodes(project) {
		for _, binding := range plant.BindingKeys(pyval.Map(pyval.Map(node, "data"), "state_mapping")) {
			add(binding)
		}
	}
	for _, entry := range pyval.List(project, "test_panel") {
		object, _ := entry.(*pyjson.Object)
		add(plant.Binding{IedID: pyval.Str(object, "ied_id"), Ref: pyval.Str(object, "ref")})
	}
	for _, binding := range bindings {
		target := c.simulatedRun(binding.IedID)
		if binding.IedID == "" || binding.Ref == "" || target == nil || (only != nil && !only[binding.IedID]) {
			continue
		}
		info, ok := target.model.lookup(binding.Ref)
		if !ok {
			continue
		}
		refs := []string{binding.Ref}
		for _, companion := range []string{info.QualityRef, info.TimestampRef} {
			if companion != "" {
				refs = append(refs, companion)
			}
		}
		values := pyjson.NewObject()
		err := target.do(func() error {
			for _, ref := range refs {
				value, err := target.worker.Read(ref)
				if err != nil {
					return err
				}
				values.Set(ref, value)
			}
			return nil
		})
		if err != nil {
			return err
		}
		for _, ref := range values.Keys() {
			value, _ := values.Get(ref)
			target.table.set(ref, value)
		}
		quality, timestamp := any(int64(0)), any(int64(0))
		if value, ok := values.Get(info.QualityRef); ok && info.QualityRef != "" {
			quality = value
		}
		if value, ok := values.Get(info.TimestampRef); ok && info.TimestampRef != "" {
			timestamp = value
		}
		value, _ := values.Get(binding.Ref)
		c.emit("signal", pyval.Obj("ied_id", binding.IedID, "ref", binding.Ref, "value", value, "quality", quality,
			"source_timestamp", timestamp, "source", "initial_native_snapshot", "updates", values))
	}
	c.plant.publishAll()
	return nil
}

// applyInitialStates writes each mapped CB/DS/ES start position (initial_state.py).
func (c *Coordinator) applyInitialStates(project *pyjson.Object, only map[string]bool) error {
	var order []string
	planned := map[string][]item{}
	for _, node := range diagramNodes(project) {
		data := pyval.Map(node, "data")
		if !plant.Switches[pyval.Str(data, "kind")] {
			continue
		}
		action := "open"
		if getOr(data, "initial_position", "open") == "closed" {
			action = "close"
		}
		targets, err := plant.FeedbackTargets(data, pyval.Map(data, "state_mapping"), action)
		if err != nil {
			return err
		}
		for _, target := range targets {
			if _, seen := planned[target.Binding.IedID]; !seen {
				order = append(order, target.Binding.IedID)
			}
			planned[target.Binding.IedID] = append(planned[target.Binding.IedID], item{target.Binding.Ref, target.Value, nil})
		}
	}
	for _, iedID := range order {
		if c.simulatedRun(iedID) == nil || (only != nil && !only[iedID]) {
			continue
		}
		if _, err := c.apply(iedID, planned[iedID], "initial_state"); err != nil {
			return err
		}
	}
	return nil
}
