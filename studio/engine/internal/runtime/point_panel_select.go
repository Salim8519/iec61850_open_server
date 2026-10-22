package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// Select before operate on a point panel row (ADR-0108): POST /equipment/<node>/test with
// {"action":"select","row":..,"value":..} holds a selection for that value, "send" with the same
// value operates within the window, {"action":"cancel","row":..} ends it.

// pointSelection answers the select and cancel actions of one command row.
func (c *Coordinator) pointSelection(nodeID string, ied *pyjson.Object, point panelPoint, action string,
	value any) (*pyjson.Object, error) {
	key := selectionKey{node: nodeID, point: point.id}
	iedID := pyval.Str(ied, "id")
	request := pyval.Obj("node_id", nodeID, "point_id", point.id, "ied_id", iedID, "ref", point.ref,
		"point_type", point.kind, "correlation_id", newHexID(), "origin", "studio_api")
	if action == "cancel" {
		held, found := c.selections.drop(key)
		if found {
			if err := c.CancelTyped(held.iedID, held.ref); err != nil {
				return nil, err
			}
			c.emit("point_panel_select_cancelled", request)
		}
		return copyWith(request, "action", action, "row", point.id, "cancelled", found), nil
	}
	if refusal := commandRefusal(ied); refusal != "" {
		return nil, apperr.ValueError(refusal)
	}
	control := typedControl(ied, point.ref)
	if control == nil || !typedCDCs[pyval.Str(control, "cdc")] {
		return nil, apperr.New(point.label + " uses a command this IED does not have: " + point.ref)
	}
	typed, err := typedValue(control, value, point.label)
	if err != nil {
		return nil, err
	}
	model := controlModelName(control)
	if !isSboModel(model) {
		return nil, apperr.New(point.label + " is operated directly (" + model + "), so it needs no Select: send it.")
	}
	answer, err := c.SelectTyped(TypedCommand{IEDID: iedID, Ref: point.ref, Value: typed})
	if err != nil {
		c.emit("point_panel_failed", copyWith(request, "value", typed, "message", point.label+": "+errorText(err)))
		return nil, err
	}
	held := c.selections.hold(key, selection{iedID: iedID, ref: point.ref, value: typed, model: answer.Model,
		windowMs: operateWindow(answer.ExpiresMs)})
	c.emit("point_panel_selected", copyWith(request, "value", typed, "expires_ms", int64(held.windowMs)))
	return copyWith(request, "action", action, "row", point.id, "value", typed, "model", answer.Model,
		"selected", true, "expires_ms", int64(held.windowMs)), nil
}

// sendPoint sends a checked value: through a held selection (same value only), today's canvas
// path for a Boolean SPC/DPC, or the typed Operate for every other class.
func (c *Coordinator) sendPoint(nodeID string, ied *pyjson.Object, point panelPoint, control *pyjson.Object, typed any, state string,
	run *running) (*pyjson.Object, error) {
	key := selectionKey{node: nodeID, point: point.id}
	iedID := pyval.Str(ied, "id")
	held, status := c.selections.peek(key)
	switch status {
	case selectionExpired:
		return nil, apperr.New("The selection of " + point.label + " ran out after " + seconds(held.windowMs) +
			", so nothing was sent. Select it again.")
	case selectionLive:
		if !pyval.Equal(held.value, typed) {
			return nil, apperr.New(point.label + " is selected for " + valueWords(held.value) + ". Operate sends " +
				valueWords(held.value) + "; Cancel first to send another value.")
		}
		c.selections.drop(key)
		return c.OperateTyped(TypedCommand{IEDID: iedID, Ref: point.ref, Value: typed})
	}
	cdc := pyval.Str(control, "cdc")
	if flag, isBool := typed.(bool); isBool && (cdc == "SPC" || cdc == "DPC") {
		return c.commands.operate(ied, state, point.ref, flag, run.host, run.port)
	}
	return c.OperateTyped(TypedCommand{IEDID: iedID, Ref: point.ref, Value: typed})
}

// valueWords is how a selected value reads in a message.
func valueWords(value any) string {
	switch typed := value.(type) {
	case string:
		return "\"" + typed + "\""
	case float64:
		return numberWords(typed)
	case bool:
		if typed {
			return "true (on / close)"
		}
		return "false (off / open)"
	}
	return pyval.Repr(value)
}
