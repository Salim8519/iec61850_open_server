package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/runtime/tapchanger"
)

// tapMode switches a transformer to Auto or Manual. With an Auto/Manual command it sends that
// command (SPC: true = Auto; ENC: auto_value or manual_value through the typed path) and, for a
// simulated status the command does not move itself, writes the status as control feedback.
// Without one, the mode is the simulated regulator's own (kept for this run only).
func (c *Coordinator) tapMode(node tapNode, auto bool) (*pyjson.Object, error) {
	action, correlation := "manual", newHexID()
	if auto {
		action = "auto"
	}
	if !node.modeCtrl.Set() {
		if node.mode.Set() {
			return nil, apperr.New("Auto/Manual of " + node.label + " comes from its IED, and no Auto/Manual command " +
				"is chosen. Choose the command in the properties while the IEDs are stopped.")
		}
		if auto && !node.settings.Enabled {
			return nil, apperr.New("Auto needs a voltage regulator. Turn on the simulated voltage regulator in the " +
				"properties while the IEDs are stopped, or choose an Auto/Manual command of the IED.")
		}
		c.taps.setAuto(node, auto, correlation)
		return pyval.Obj("node_id", node.id, "action", action, "wire_value", nil, "correlation_id", correlation), nil
	}
	ied := findIed(c.opt.Project.Current(), node.modeCtrl.IedID)
	if ied == nil {
		return nil, apperr.Coded("Unknown IED", "not_found")
	}
	control := typedControl(ied, node.modeCtrl.Ref)
	request := pyval.Obj("node_id", node.id, "ied_id", node.modeCtrl.IedID, "ref", node.modeCtrl.Ref,
		"action", action, "correlation_id", correlation, "origin", "studio_api")
	c.emit("tap_command_requested", request)
	var wire any = auto
	var answer *pyjson.Object
	var err error
	if pyval.Str(control, "cdc") == "ENC" {
		wire = node.manualValue
		if auto {
			wire = node.autoValue
		}
		if refusal := commandRefusal(ied); refusal != "" {
			err = apperr.ValueError(refusal)
		} else {
			answer, err = c.OperateTyped(TypedCommand{IEDID: node.modeCtrl.IedID, Ref: node.modeCtrl.Ref, Value: wire})
		}
	} else {
		answer, err = c.sendSpc(ied, node.modeCtrl.Ref, auto)
	}
	if err != nil {
		c.emit("tap_command_failed", copyWith(request, "message", node.label+": "+errorText(err)))
		return nil, err
	}
	if node.mode.Set() && c.simulatedRun(node.mode.IedID) != nil && node.mode != node.modeCtrlStatus(control) {
		if _, err := c.apply(node.mode.IedID, []item{{node.mode.Ref, c.modeWire(node, auto), nil}}, "control_feedback"); err != nil {
			c.emit("tap_command_failed", copyWith(request, "message", node.label+": "+errorText(err)))
			return nil, apperr.RuntimeError("The command was sent, but the simulated Auto/Manual status could not be written")
		}
	}
	c.emit("tap_command_applied", copyWith(request, "message", node.label, "wire_value", wire))
	return pyval.Obj("node_id", node.id, "action", action, "wire_value", wire, "command", answer,
		"correlation_id", correlation), nil
}

// modeCtrlStatus is the status point the Auto/Manual control moves itself.
func (n tapNode) modeCtrlStatus(control *pyjson.Object) tapchanger.Binding {
	return tapchanger.Binding{IedID: n.modeCtrl.IedID, Ref: pyval.Str(control, "status")}
}

// modeWire is the value the mode status holds for Auto or Manual: true/false for a Boolean
// status, auto_value/manual_value for a whole-number one.
func (c *Coordinator) modeWire(node tapNode, auto bool) any {
	if target := c.simulatedRun(node.mode.IedID); target != nil {
		if info, found := target.model.lookup(node.mode.Ref); found && info.Kind == "BOOLEAN" {
			return auto
		}
	}
	if auto {
		return node.autoValue
	}
	return node.manualValue
}

// modeOf reads Auto from a mode status value: true, or a number equal to auto_value.
func modeOf(node tapNode, value any) (auto, known bool) {
	if flag, ok := value.(bool); ok {
		return flag, true
	}
	if number, ok := wholeNumber(value); ok {
		return number == node.autoValue, true
	}
	return false, false
}
