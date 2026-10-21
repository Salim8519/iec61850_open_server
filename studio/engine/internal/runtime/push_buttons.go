package runtime

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime/plant"
	"studio-engine/internal/runtime/pyval"
)

// buttonPoint checks a push button's direct Boolean state point (EquipmentControl._toggle and
// _momentary_status share these checks; only the last text differs).
func (c *Coordinator) buttonPoint(node *pyjson.Object, action, kindText string) (plant.Binding, any, error) {
	if action != "" && action != "press" {
		return plant.Binding{}, nil, apperr.ValueError("Action is incompatible with this equipment")
	}
	mapping := pyval.Map(pyval.Map(node, "data"), "state_mapping")
	if mapping == nil || mapping.Len() == 0 || pyval.Get(mapping, "mode") != "direct" {
		return plant.Binding{}, nil, apperr.ValueError("Assign a Boolean state point before pressing this button")
	}
	binding := plant.BindingOf(pyval.Map(mapping, "source"))
	item, err := c.validateTarget(binding)
	if err != nil {
		return binding, nil, err
	}
	info, _ := item.model.lookup(binding.Ref)
	value, _ := item.table.get(binding.Ref)
	if _, isBool := value.(bool); info.Kind != "BOOLEAN" || !isBool {
		return binding, nil, apperr.ValueError(kindText)
	}
	return binding, value, nil
}

func pressResult(nodeID string, binding plant.Binding, value any) *pyjson.Object {
	return pyval.Obj("node_id", nodeID, "action", "press", "wire_value", nil,
		"feedback", []any{pyval.Obj("ied_id", binding.IedID, "ref", binding.Ref, "value", value)})
}

// momentary is _momentary_status: a self-resetting button is a backend pulse.
func (c *Coordinator) momentary(node *pyjson.Object, action string) (*pyjson.Object, error) {
	binding, _, err := c.buttonPoint(node, action, "Self-resetting push button requires a Boolean state point")
	if err != nil {
		return nil, err
	}
	data := pyval.Map(node, "data")
	rest := pyval.Truthy(getOr(data, "push_rest_value", false))
	duration := getOr(data, "push_reset_ms", int64(1000))
	if !pyval.IsInt(duration) {
		return nil, apperr.ValueError("Self-resetting push button requires a reset delay in milliseconds")
	}
	started, err := c.pulses.start(binding.IedID, binding.Ref, !rest, duration)
	if err != nil {
		return nil, err
	}
	position := "open"
	if rest {
		position = "closed"
	}
	nodeID := pyval.Str(node, "id")
	if err := c.opt.Project.NodeData(nodeID, pyval.Obj("initial_position", position)); err != nil {
		return nil, err
	}
	value := pyval.Get(started, "value")
	c.emit("equipment_button_applied", pyval.Obj("node_id", nodeID, "action", "press", "value", value))
	return pressResult(nodeID, binding, value), nil
}

// toggle is _toggle: a latching button flips its Boolean point.
func (c *Coordinator) toggle(node *pyjson.Object, action string) (*pyjson.Object, error) {
	binding, current, err := c.buttonPoint(node, action, "Latching push button requires a Boolean state point")
	if err != nil {
		return nil, err
	}
	value := !current.(bool)
	result, err := c.apply(binding.IedID, []item{{binding.Ref, value, nil}}, "equipment_button")
	if err != nil {
		return nil, err
	}
	actual, ok := result.Get(binding.Ref)
	if !ok {
		return nil, apperr.KeyError(binding.Ref)
	}
	position := "open"
	if value {
		position = "closed"
	}
	nodeID := pyval.Str(node, "id")
	if err := c.opt.Project.NodeData(nodeID, pyval.Obj("initial_position", position)); err != nil {
		return nil, err
	}
	c.emit("equipment_button_applied", pyval.Obj("node_id", nodeID, "action", "press", "value", actual))
	return pressResult(nodeID, binding, actual), nil
}
