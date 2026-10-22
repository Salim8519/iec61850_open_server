package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/pyval"
	"studio-engine/internal/runtime/tapchanger"
)

// The tap-changer transformer (ADR-0109): a Go-only diagram kind whose data the project package
// validates. While running, its tap position, Auto/Manual status and LV voltage are published as
// equipment_state rows "<node>:tap", "<node>:mode" and "<node>:lv" (like relay rows), and its
// regulator state as "<node>:avr". Raise, Lower, Auto and Manual arrive through the existing
// POST /equipment/<node>/operate (action raise, lower, auto, manual).

const tapTransformerKind = "TAP_TRANSFORMER"

// tapNode is one transformer as the runtime needs it.
type tapNode struct {
	id, label                    string
	settings                     tapchanger.Settings
	commands                     tapchanger.Commands
	tap, mode, voltage, modeCtrl tapchanger.Binding
	autoValue, manualValue       int64
}

func tapBinding(object *pyjson.Object, key string) tapchanger.Binding {
	binding := pyval.Map(object, key)
	return tapchanger.Binding{IedID: pyval.Str(binding, "ied_id"), Ref: pyval.Str(binding, "ref")}
}

func readTapNode(node *pyjson.Object) tapNode {
	data := pyval.Map(node, "data")
	tap := pyval.Map(data, "tap_changer")
	whole := func(key string, fallback int64) int64 {
		if value, ok := wholeNumber(pyval.Get(tap, key)); ok {
			return value
		}
		return fallback
	}
	return tapNode{id: pyval.Str(node, "id"), label: pyval.Str(data, "label"), settings: tapchanger.Read(tap),
		commands: tapchanger.Commands{Mode: pyval.Str(tap, "command_mode"), Step: tapBinding(tap, "step_control"),
			Raise: tapBinding(tap, "raise_control"), Lower: tapBinding(tap, "lower_control")},
		tap: tapBinding(tap, "tap_source"), mode: tapBinding(tap, "mode_source"),
		voltage: tapBinding(tap, "voltage_source"), modeCtrl: tapBinding(tap, "mode_control"),
		autoValue: whole("auto_value", 1), manualValue: whole("manual_value", 0)}
}

// commandIED is the IED the raise/lower commands go to ("" when none).
func (n tapNode) commandIED() string {
	if n.commands.Mode == "step" {
		return n.commands.Step.IedID
	}
	return n.commands.Raise.IedID
}

// tapNodes are the transformers of a diagram.
func tapNodes(diagram *pyjson.Object) []tapNode {
	var nodes []tapNode
	for _, item := range pyval.List(diagram, "nodes") {
		node, ok := item.(*pyjson.Object)
		if ok && pyval.Str(pyval.Map(node, "data"), "kind") == tapTransformerKind {
			nodes = append(nodes, readTapNode(node))
		}
	}
	return nodes
}

// addTapChangerRows adds the live rows of a transformer to the plant states.
func addTapChangerRows(nodeID string, data *pyjson.Object, add func(string, *pyjson.Object)) {
	tap := pyval.Map(data, "tap_changer")
	for _, row := range [][2]string{{"tap", "tap_source"}, {"mode", "mode_source"}, {"lv", "voltage_source"}} {
		if source := pyval.Map(tap, row[1]); source != nil && source.Len() > 0 {
			add(nodeID+":"+row[0], pyval.Obj("mode", "direct", "source", source))
		}
	}
}

// readTap is the present tap position and whether it is known.
func (c *Coordinator) readTap(node tapNode) (int64, bool) {
	if !node.tap.Set() {
		return 0, false
	}
	value := c.plant.read(bindingOf(node.tap))
	if pyval.IsBool(value) {
		return 0, false
	}
	return wholeNumber(value)
}

// tapOperate is POST /equipment/<node>/operate for a transformer; the caller checked the run.
func (c *Coordinator) tapOperate(node *pyjson.Object, action string) (*pyjson.Object, error) {
	item := readTapNode(node)
	switch action {
	case tapchanger.Raise, tapchanger.Lower:
		if auto, known := c.taps.autoMode(item); known && auto {
			return nil, apperr.New(item.label + " is in Auto: the voltage regulator moves the tap itself. " +
				"Switch to Manual first, then raise or lower.")
		}
		return c.tapStep(item, action, "studio_api", newHexID())
	case "auto", "manual":
		return c.tapMode(item, action == "auto")
	}
	return nil, apperr.New("Unknown transformer action. Use raise, lower, auto or manual.")
}
