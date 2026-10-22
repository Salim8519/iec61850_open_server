package runtime

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/plant"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/runtime/tapchanger"
)

func bindingOf(binding tapchanger.Binding) plant.Binding {
	return plant.Binding{IedID: binding.IedID, Ref: binding.Ref}
}

// tapStep sends one raise or lower: a BSC/ISC through OperateTyped, an SPC pair through the
// canvas command path. Then, for a simulated tap position the command itself does not move, it
// writes the new tap as control feedback. origin is "studio_api" or "avr"; every step is logged
// as requested, then applied or failed, with the same correlation id.
func (c *Coordinator) tapStep(node tapNode, action, origin, correlation string) (*pyjson.Object, error) {
	ied := findIed(c.opt.Project.Current(), node.commandIED())
	if node.commandIED() != "" && ied == nil {
		return nil, apperr.Coded("Unknown IED", "not_found")
	}
	if ied != nil {
		if control := typedControl(ied, node.commands.Step.Ref); control != nil {
			node.commands.StepCDC = pyval.Str(control, "cdc")
		}
	}
	tap, known := c.readTap(node)
	command, err := tapchanger.Route(node.settings, node.commands, action, tap, known)
	if err != nil {
		return nil, apperr.New(err.Error())
	}
	request := pyval.Obj("node_id", node.id, "ied_id", command.Target.IedID, "ref", command.Target.Ref,
		"action", action, "tap", knownTap(tap, known), "correlation_id", correlation, "origin", origin)
	c.emit("tap_command_requested", request)
	fail := func(err error) (*pyjson.Object, error) {
		c.emit("tap_command_failed", copyWith(request, "message", node.label+": "+errorText(err)))
		return nil, err
	}
	var answer *pyjson.Object
	if command.Typed {
		if refusal := commandRefusal(ied); refusal != "" {
			return fail(apperr.ValueError(refusal))
		}
		answer, err = c.OperateTyped(TypedCommand{IEDID: command.Target.IedID, Ref: command.Target.Ref, Value: command.Value})
	} else {
		answer, err = c.sendSpc(ied, command.Target.Ref, true)
	}
	if err != nil {
		return fail(err)
	}
	feedback := false
	if known && c.tapNeedsFeedback(node, ied) {
		if _, err := c.apply(node.tap.IedID, []item{{node.tap.Ref, command.Next, nil}}, "control_feedback"); err != nil {
			return fail(apperr.RuntimeError("The command was sent, but the simulated tap position could not be " +
				"written: " + errorText(err)))
		}
		feedback = true
	}
	after, afterKnown := c.readTap(node)
	c.emit("tap_command_applied", copyWith(request, "message", node.label, "tap_after", knownTap(after, afterKnown),
		"feedback", feedback))
	return pyval.Obj("node_id", node.id, "action", action, "wire_value", true, "command", answer,
		"tap", knownTap(after, afterKnown), "feedback", feedback, "correlation_id", correlation), nil
}

func knownTap(tap int64, known bool) any {
	if !known {
		return nil
	}
	return tap
}

// tapNeedsFeedback says whether ScadaBench must write the new tap: the tap position is on a
// running simulated IED and is not the status of the step control (a BSC/ISC moves its own
// valWTr/stVal in the native server).
func (c *Coordinator) tapNeedsFeedback(node tapNode, ied *pyjson.Object) bool {
	if c.simulatedRun(node.tap.IedID) == nil {
		return false
	}
	if node.commands.Mode == "step" && ied != nil && node.tap.IedID == node.commands.Step.IedID {
		if control := typedControl(ied, node.commands.Step.Ref); control != nil {
			status := pyval.Str(control, "status")
			if status != "" && (node.tap.Ref == status || strings.HasPrefix(node.tap.Ref, status+".")) {
				return false
			}
		}
	}
	return true
}

// sendSpc operates one SPC with value through the canvas command path (Operate): the same MMS
// client, the same refusal of real devices, the command marked as ScadaBench's own.
func (c *Coordinator) sendSpc(ied *pyjson.Object, ref string, value bool) (*pyjson.Object, error) {
	if ied == nil {
		return nil, apperr.Coded("Unknown IED", "not_found")
	}
	if refusal := commandRefusal(ied); refusal != "" {
		return nil, apperr.ValueError(refusal)
	}
	control := newIedModel(ied).control(ref)
	if control == nil || pyval.Get(control, "cdc") != "SPC" {
		return nil, apperr.New("This IED has no single command (SPC) " + ref + ". Choose the command again.")
	}
	iedID := pyval.Str(ied, "id")
	c.commandMu.Lock()
	defer c.commandMu.Unlock()
	c.access.markOwn(iedID, ref)
	c.mu.Lock()
	run, state := c.runs[iedID], c.iedStates[iedID]
	c.mu.Unlock()
	if run == nil || run.worker == nil {
		return nil, apperr.ValueError("IED is not running")
	}
	return c.commands.operate(ied, state, ref, value, run.host, run.port)
}
