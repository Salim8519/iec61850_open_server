package runtime

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
)

// The point panel (ADR-0102): a Go-only diagram kind with one IED and up to 300 points. The
// project package validates its data; this file reads it while running. Status, double and
// value points write the simulated IED like a test button (no command is sent); command points
// go through the canvas command path (Operate). Both arrive through POST /equipment/<node>/test
// (actions set, send, healthy; select and cancel for SBO commands, ADR-0108), so the Flask
// route table does not change.

const pointPanelKind = "POINT_PANEL"

// panelBatch is how many points one native write carries: each point may add a quality and a
// time attribute; 64 points stay well inside one native batch (nativeBatchLimit).
const panelBatch = 64

const panelRealOnly = "{name} is a real device. A point panel only changes IEDs that ScadaBench simulates; " +
	"operate real equipment from Tools."

// panelPoint is one row of a panel as the runtime needs it.
type panelPoint struct {
	id, label, kind, ref string
	healthy              any // nil when the point has none
}

// panelPoints reads the rows of a panel's node data: status, double and value points name a
// signal, command points a control.
func panelPoints(data *pyjson.Object) []panelPoint {
	var points []panelPoint
	for _, item := range pyval.List(data, "points") {
		row, ok := item.(*pyjson.Object)
		if !ok {
			continue
		}
		point := panelPoint{id: pyval.Str(row, "id"), label: pyval.Str(row, "label"), kind: pyval.Str(row, "type"),
			healthy: pyval.Get(row, "healthy")}
		if point.kind == "command" {
			point.ref = pyval.Str(pyval.Map(row, "control"), "ref")
		} else {
			point.ref = pyval.Str(pyval.Map(row, "source"), "ref")
		}
		points = append(points, point)
	}
	return points
}

// addPointPanelRows adds the live-value mapping of every point that reads a signal: its raw
// value is published as equipment_state "<node>:<point>", like relay and measurement rows.
func addPointPanelRows(nodeID string, data *pyjson.Object, add func(string, *pyjson.Object)) {
	for _, item := range pyval.List(data, "points") {
		row, ok := item.(*pyjson.Object)
		if !ok {
			continue
		}
		if source := pyval.Map(row, "source"); source != nil && source.Len() > 0 {
			add(nodeID+":"+pyval.Str(row, "id"), pyval.Obj("mode", "direct", "source", source))
		}
	}
}

func findPoint(points []panelPoint, id any) (panelPoint, bool) {
	name, _ := id.(string)
	for _, point := range points {
		if name != "" && point.id == name {
			return point, true
		}
	}
	return panelPoint{}, false
}

// pointPanelRequest is POST /equipment/<node>/test for a point panel: action "set" writes one
// status, double or value point, "send" sends one command point, "healthy" writes every point
// that has a healthy value. The caller found the node and checked its kind.
func (c *Coordinator) pointPanelRequest(nodeID string, node *pyjson.Object, action, value, row any) (*pyjson.Object, error) {
	data := pyval.Map(node, "data")
	if !c.isLive() {
		return nil, apperr.New("Run the IEDs to change points. Press Run, then try again.")
	}
	iedID := pyval.Str(data, "ied_id")
	if iedID == "" {
		return nil, apperr.New("This point panel has no IED yet. Choose one in its properties while the IEDs are stopped.")
	}
	ied := findIed(c.opt.Project.Current(), iedID)
	if ied == nil {
		return nil, apperr.Coded("Unknown IED", "not_found")
	}
	points := panelPoints(data)
	name, _ := action.(string)
	if name == "healthy" {
		return c.pointPanelHealthy(nodeID, ied, points)
	}
	if name != "set" && name != "send" && name != "select" && name != "cancel" {
		return nil, apperr.New("Unknown point action. Use set, send, select, cancel or healthy.")
	}
	point, found := findPoint(points, row)
	if !found {
		return nil, apperr.New("This panel has no such point. Choose a point of the panel.")
	}
	if name == "select" || name == "cancel" {
		if point.kind != "command" {
			return nil, apperr.New(point.label + " is not a command: set its value, it cannot be selected.")
		}
		return c.pointSelection(nodeID, ied, point, name, value)
	}
	if (name == "send") != (point.kind == "command") {
		if point.kind == "command" {
			return nil, apperr.New(point.label + " is a command: send it, it cannot be set.")
		}
		return nil, apperr.New(point.label + " is not a command: set its value, it cannot be sent.")
	}
	correlation := newHexID()
	request := pyval.Obj("node_id", nodeID, "point_id", point.id, "ied_id", iedID, "ref", point.ref,
		"point_type", point.kind, "value", value, "correlation_id", correlation, "origin", "studio_api")
	c.emit("point_panel_requested", request)
	var result *pyjson.Object
	var err error
	if name == "send" {
		result, err = c.pointSend(nodeID, ied, point, value)
	} else {
		result, err = c.pointSet(ied, point, value)
	}
	if err != nil {
		failed := copyWith(request, "message", point.label+": "+errorText(err))
		c.emit("point_panel_failed", failed)
		return nil, err
	}
	applied, _ := result.Get("value")
	c.emit("point_panel_applied", copyWith(request, "message", point.label, "value", applied))
	return copyWith(result, "node_id", nodeID, "action", name, "row", point.id, "type", point.kind,
		"ied_id", iedID, "correlation_id", correlation), nil
}

// copyWith returns a copy of object with the given key and value pairs set.
func copyWith(object *pyjson.Object, pairs ...any) *pyjson.Object {
	result := pyval.Copy(object).(*pyjson.Object)
	for index := 0; index+1 < len(pairs); index += 2 {
		result.Set(pairs[index].(string), pairs[index+1])
	}
	return result
}

// panelTarget is the running simulated IED a point write needs, or a plain-words refusal.
func (c *Coordinator) panelTarget(ied *pyjson.Object) (*running, error) {
	name := pyval.Str(ied, "name")
	switch roleOf(ied) {
	case "real":
		return nil, apperr.New(strings.Replace(panelRealOnly, "{name}", name, 1))
	case "off":
		return nil, apperr.New(name + " is skipped in this project's run. Set it to 'Simulate' to change its points.")
	}
	target := c.simulatedRun(pyval.Str(ied, "id"))
	if target == nil {
		return nil, apperr.New(name + " is not running. Run this IED first.")
	}
	return target, nil
}
