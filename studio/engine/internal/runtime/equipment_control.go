package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/plant"
	"studio-engine/internal/runtime/pyval"
)

// findNode is ProjectService.find_node.
func (c *Coordinator) findNode(nodeID string) (*pyjson.Object, error) {
	for _, node := range diagramNodes(c.opt.Project.Current()) {
		if pyval.Str(node, "id") == nodeID {
			return node, nil
		}
	}
	return nil, apperr.Coded("Unknown equipment", "not_found")
}

// validateTarget is EquipmentControl._validate_target.
func (c *Coordinator) validateTarget(binding plant.Binding) (*running, error) {
	c.mu.Lock()
	item := c.runs[binding.IedID]
	c.mu.Unlock()
	if item != nil {
		if info, ok := item.model.lookup(binding.Ref); ok && info.Writable && info.FC != "CF" && item.worker != nil {
			return item, nil
		}
	}
	return nil, apperr.ValueError("Mapped equipment state point cannot be updated while running")
}

// Operate is POST /equipment/<node>/operate (EquipmentControl.operate). action "" is None.
func (c *Coordinator) Operate(nodeID string, value any, action string) (*pyjson.Object, error) {
	if !c.isLive() {
		return nil, apperr.ValueError("Run the IEDs before operating equipment")
	}
	node, err := c.findNode(nodeID)
	if err != nil {
		return nil, err
	}
	data := pyval.Map(node, "data")
	switch {
	case pyval.Str(data, "kind") == "PUSH_BUTTON" && pyval.Get(data, "push_mode") == "latching_status":
		return c.toggle(node, action)
	case pyval.Str(data, "kind") == "PUSH_BUTTON" && pyval.Get(data, "push_mode") == "self_resetting":
		return c.momentary(node, action)
	}
	semantic, wire, err := plant.SemanticCommand(data, action, value)
	if err != nil {
		return nil, err
	}
	binding := pyval.Map(data, "control_binding")
	if binding == nil || binding.Len() == 0 {
		return nil, apperr.ValueError("Equipment has no command mapping")
	}
	ied := findIed(c.opt.Project.Current(), pyval.Str(binding, "ied_id"))
	if ied == nil {
		return nil, apperr.Coded("Unknown IED", "not_found")
	}
	if refusal := commandRefusal(ied); refusal != "" {
		return nil, apperr.ValueError(refusal)
	}
	control := newIedModel(ied).control(pyval.Str(binding, "ref"))
	expected, cdcErr := plant.ExpectedCDC(pyval.Str(data, "kind"))
	if control == nil || cdcErr != nil || pyval.Get(control, "cdc") != expected {
		if control != nil && cdcErr != nil {
			return nil, cdcErr
		}
		return nil, apperr.ValueError("Equipment command mapping is incompatible")
	}
	targets, err := plant.FeedbackTargets(data, pyval.Map(data, "state_mapping"), semantic)
	if err != nil {
		return nil, err
	}
	iedID, statusRef := pyval.Str(ied, "id"), pyval.Str(control, "status")
	var planned []plant.Target
	for _, target := range targets {
		if target.Binding != (plant.Binding{IedID: iedID, Ref: statusRef}) || !pyval.Equal(target.Value, wire) {
			planned = append(planned, target)
		}
	}
	for _, target := range planned {
		if _, err := c.validateTarget(target.Binding); err != nil {
			return nil, err
		}
	}
	return c.command(nodeID, data, ied, control, planned, semantic, wire)
}

func (c *Coordinator) command(nodeID string, data, ied, control *pyjson.Object, planned []plant.Target,
	semantic string, wire bool) (*pyjson.Object, error) {
	c.commandMu.Lock()
	defer c.commandMu.Unlock()
	iedID, controlRef := pyval.Str(ied, "id"), pyval.Str(control, "ref")
	c.plant.markOwn(nodeID)
	c.access.markOwn(iedID, controlRef)
	c.mu.Lock()
	item, state := c.runs[iedID], c.iedStates[iedID]
	c.mu.Unlock()
	if item == nil || item.worker == nil {
		return nil, apperr.ValueError("IED is not running")
	}
	result, err := c.commands.operate(ied, state, controlRef, wire, item.host, item.port)
	if err != nil {
		return nil, err
	}
	applied, changed, err := c.applyFeedback(planned)
	if err != nil {
		c.emit("equipment_feedback_failed", pyval.Obj("node_id", nodeID, "control_ref", controlRef, "message", errorText(err)))
		return nil, apperr.RuntimeError("MMS command was applied, but simulated equipment feedback failed")
	}
	c.emit("equipment_feedback_applied", pyval.Obj("node_id", nodeID, "control_ref", controlRef,
		"target_count", int64(len(applied)), "changed_count", changed, "action", semantic, "wire_value", wire,
		"origin", "studio_api"))
	position := "open"
	if plant.Switches[pyval.Str(data, "kind")] && semantic == "close" {
		position = "closed"
	}
	if err := c.opt.Project.NodeData(nodeID, pyval.Obj("initial_position", position)); err != nil {
		return nil, err
	}
	c.plant.publishNode(nodeID)
	return pyval.Obj("command", result, "feedback", applied, "node_id", nodeID, "action", semantic,
		"wire_value", wire), nil
}

// applyFeedback is EquipmentControl._apply: write the planned feedback per IED.
func (c *Coordinator) applyFeedback(planned []plant.Target) ([]any, int64, error) {
	previous := map[plant.Binding]any{}
	var order []string
	grouped := map[string][]item{}
	for _, target := range planned {
		previous[target.Binding] = c.plant.read(target.Binding)
		if _, seen := grouped[target.Binding.IedID]; !seen {
			order = append(order, target.Binding.IedID)
		}
		grouped[target.Binding.IedID] = append(grouped[target.Binding.IedID], item{target.Binding.Ref, target.Value, nil})
	}
	applied := []any{}
	changed := int64(0)
	for _, iedID := range order {
		result, err := c.apply(iedID, grouped[iedID], "control_feedback")
		if err != nil {
			return nil, 0, err
		}
		for _, written := range grouped[iedID] {
			value, ok := result.Get(written.ref)
			if !ok {
				return nil, 0, apperr.KeyError(written.ref)
			}
			applied = append(applied, pyval.Obj("ied_id", iedID, "ref", written.ref, "value", value))
			if !pyval.Equal(previous[plant.Binding{IedID: iedID, Ref: written.ref}], value) {
				changed++
			}
		}
	}
	return applied, changed, nil
}

// commandRefusal is network_settings.command_refusal ("" when ScadaBench may command the IED).
func commandRefusal(ied *pyjson.Object) string {
	name := pyval.Str(ied, "name")
	switch roleOf(ied) {
	case "real":
		return name + " is a real device, so nothing was sent from here. Real equipment is " +
			"operated only from Tools (IED Explorer or the gateway), after a confirmation."
	case "off":
		return name + " is skipped in this project's run. Set it to 'Simulate' to operate it."
	}
	return ""
}
