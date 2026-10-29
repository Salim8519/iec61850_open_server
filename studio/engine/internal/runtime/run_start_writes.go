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
	for _, binding := range goOnlyRowBindings(project) {
		add(binding)
	}
	var plan []*snapshotRead
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
		plan = append(plan, &snapshotRead{binding: binding, target: target, qualityRef: info.QualityRef,
			stampRef: info.TimestampRef, refs: refs})
	}
	readSnapshots(plan) // all IEDs at once, each IED's reads overlapped (ADR-0129)
	for _, read := range plan {
		if err := read.failure(); err != nil {
			return err
		}
		values := pyjson.NewObject()
		for index, ref := range read.refs {
			values.Set(ref, read.values[index])
		}
		for _, ref := range values.Keys() {
			value, _ := values.Get(ref)
			read.target.table.set(ref, value)
		}
		quality, timestamp := any(int64(0)), any(int64(0))
		if value, ok := values.Get(read.qualityRef); ok && read.qualityRef != "" {
			quality = value
		}
		if value, ok := values.Get(read.stampRef); ok && read.stampRef != "" {
			timestamp = value
		}
		value, _ := values.Get(read.binding.Ref)
		c.emit("signal", pyval.Obj("ied_id", read.binding.IedID, "ref", read.binding.Ref, "value", value, "quality", quality,
			"source_timestamp", timestamp, "source", "initial_native_snapshot", "updates", values))
	}
	c.plant.publishAll()
	return nil
}

// goOnlyRowBindings are the sources of the Go-only SLD rows (tap changer, point panel and
// measurement list, change 0176). A worker restarts from its SCL values, so these rows are read
// back at Run like the mapped states; otherwise the SLD keeps the previous run's values. Relay
// statuses stay out because the frozen Python engine does not read them back either (parity).
func goOnlyRowBindings(project *pyjson.Object) []plant.Binding {
	var bindings []plant.Binding
	collect := func(_ string, mapping *pyjson.Object) { bindings = append(bindings, plant.BindingKeys(mapping)...) }
	for _, node := range diagramNodes(project) {
		data := pyval.Map(node, "data")
		switch pyval.Str(data, "kind") {
		case pointPanelKind:
			addPointPanelRows(pyval.Str(node, "id"), data, collect)
		case tapTransformerKind:
			addTapChangerRows(pyval.Str(node, "id"), data, collect)
		case "MEASUREMENT_LIST":
			for _, entry := range pyval.List(data, "measurements") {
				row, _ := entry.(*pyjson.Object)
				if source := pyval.Map(row, "source"); source != nil && source.Len() > 0 {
					collect("", pyval.Obj("mode", "direct", "source", source))
				}
			}
		}
	}
	return bindings
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
