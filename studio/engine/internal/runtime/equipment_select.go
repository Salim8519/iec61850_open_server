package runtime

import (
	"strconv"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Select before operate for a CB, DS or ES (ADR-0108): POST /equipment/<node>/select holds a
// selection, Operate sends within its window, POST /equipment/<node>/cancel ends it. A direct
// model never selects and its Operate keeps today's exact request and path.

// EquipmentSelect selects the node's control for action "open" or "close".
func (c *Coordinator) EquipmentSelect(nodeID, action string, options any) (*pyjson.Object, error) {
	plan, err := c.switchPlan(nodeID, action)
	if err != nil {
		return nil, err
	}
	opts, sent, err := commandOptionsFrom(options)
	if err != nil {
		return nil, err
	}
	label := pyval.Str(plan.data, "label")
	model := controlModelName(plan.control)
	if !isSboModel(model) {
		return nil, apperr.ValueError(label + " is operated directly (" + model + "), so it needs no Select. Press " +
			title(plan.semantic) + " to send the command.")
	}
	iedID, ref := pyval.Str(plan.ied, "id"), pyval.Str(plan.control, "ref")
	request := pyval.Obj("node_id", nodeID, "control_ref", ref, "action", plan.semantic, "model", model, "origin", "studio_api")
	answer, err := c.SelectTyped(TypedCommand{IEDID: iedID, Ref: ref, Value: plan.wire, Options: opts})
	if err != nil {
		c.emit("equipment_select_failed", copyWith(request, "message", errorText(err)))
		return nil, err
	}
	held := c.selections.hold(selectionKey{node: nodeID}, selection{iedID: iedID, ref: ref, action: plan.semantic,
		value: plan.wire, options: opts, model: answer.Model, windowMs: operateWindow(answer.ExpiresMs)})
	if changes := optionChanges(plan.data, opts, sent); changes != nil {
		if err := c.opt.Project.NodeData(nodeID, changes); err != nil {
			return nil, err
		}
	}
	c.emit("equipment_selected", copyWith(request, "expires_ms", int64(held.windowMs)))
	return pyval.Obj("node_id", nodeID, "action", plan.semantic, "model", answer.Model, "selected", true,
		"expires_ms", int64(held.windowMs)), nil
}

// EquipmentCancel ends the node's selection; without one it answers that nothing was selected.
func (c *Coordinator) EquipmentCancel(nodeID string) (*pyjson.Object, error) {
	if _, err := c.findNode(nodeID); err != nil {
		return nil, err
	}
	held, found := c.selections.drop(selectionKey{node: nodeID})
	if !found {
		return pyval.Obj("node_id", nodeID, "cancelled", false), nil
	}
	if err := c.CancelTyped(held.iedID, held.ref); err != nil {
		return nil, err
	}
	c.emit("equipment_select_cancelled", pyval.Obj("node_id", nodeID, "control_ref", held.ref, "action", held.action,
		"origin", "studio_api"))
	return pyval.Obj("node_id", nodeID, "cancelled", true, "action", held.action), nil
}

// switchPlan is Operate's checks for a select: running, a CB/DS/ES, a valid command plan.
func (c *Coordinator) switchPlan(nodeID, action string) (commandPlan, error) {
	if !c.isLive() {
		return commandPlan{}, apperr.ValueError("Run the IEDs before operating equipment")
	}
	node, err := c.findNode(nodeID)
	if err != nil {
		return commandPlan{}, err
	}
	data := pyval.Map(node, "data")
	if kind := pyval.Str(data, "kind"); kind != "CB" && kind != "DS" && kind != "ES" {
		return commandPlan{}, apperr.ValueError("Only a breaker, disconnector or earth switch is selected before it operates")
	}
	return c.planCommand(data, nil, action)
}

// sendEquipment sends one planned command. A held selection must still hold and match the
// action; then the typed Operate carries the selected options. Without a selection, default
// options use today's canvas path unchanged; other options go through the typed Operate.
func (c *Coordinator) sendEquipment(nodeID string, plan commandPlan, options CommandOptions, state string,
	run *running) (*pyjson.Object, error) {
	key := selectionKey{node: nodeID}
	iedID, ref := pyval.Str(plan.ied, "id"), pyval.Str(plan.control, "ref")
	held, status := c.selections.peek(key)
	label := pyval.Str(plan.data, "label")
	switch status {
	case selectionExpired:
		c.emit("equipment_select_expired", pyval.Obj("node_id", nodeID, "control_ref", ref, "action", held.action))
		return nil, apperr.ValueError("The selection of " + label + " ran out after " + seconds(held.windowMs) +
			", so nothing was sent. Press " + title(held.action) + " again to select it.")
	case selectionLive:
		if held.action != plan.semantic {
			return nil, apperr.ValueError(label + " is selected to " + held.action + ". Press Operate to " + held.action +
				" it, or Cancel first.")
		}
		c.selections.drop(key)
		return c.OperateTyped(TypedCommand{IEDID: iedID, Ref: ref, Value: plan.wire, Options: held.options})
	}
	if options == (CommandOptions{}) {
		return c.commands.operate(plan.ied, state, ref, plan.wire, run.host, run.port)
	}
	return c.OperateTyped(TypedCommand{IEDID: iedID, Ref: ref, Value: plan.wire, Options: options})
}

// title is "Close" for "close".
func title(word string) string {
	if word == "" {
		return word
	}
	return strings.ToUpper(word[:1]) + word[1:]
}

// seconds is "27 s" for 27000.
func seconds(ms int) string { return strconv.Itoa((ms+500)/1000) + " s" }
