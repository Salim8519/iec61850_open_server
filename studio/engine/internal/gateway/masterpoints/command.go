package masterpoints

import (
	"strings"

	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// CommandOf is the command a point's Send button sends: its stored command type, address,
// select before operate and value, plus the DNP3 control code and the Modbus word order. It
// goes through the master's existing command route, so the same checks and confirmation apply.
// A monitored point gives nil.
func CommandOf(protocol string, point *pyjson.Object) any {
	kind := text(point, "command")
	if kind == "" {
		return nil
	}
	command := pyjson.NewObject().Set("kind", kind).Set("address", Address(point)).Set("value", field(point, "value")).
		Set("select", field(point, "select") == true)
	if code := text(point, "code"); code != "" {
		command.Set("code", code)
	}
	if strings.HasPrefix(kind, "registers_") {
		command.Set("word_order", text(point, "word_order"))
	}
	return command
}

// numericGroups are the reported groups whose value is a number (scale and offset apply).
var numericGroups = map[string]bool{"normalized": true, "scaled": true, "float": true, "counter": true,
	"analog": true, "analog_output": true, "frozen_counter": true, "holding_register": true, "input_register": true,
	"step": true}

// engineering is the received value in the point's units: raw x scale + offset for numbers,
// the value in words for on/off and positions.
func engineering(point *pyjson.Object, found Received) string {
	if !numericGroups[found.Group] {
		return text(found.Row, "value_text")
	}
	raw, ok := pytext.Float(field(found.Row, "value"))
	if !ok || field(found.Row, "value") == nil {
		return text(found.Row, "value_text")
	}
	scale, _ := pytext.Float(field(point, "scale"))
	offset, _ := pytext.Float(field(point, "offset"))
	value := pytext.FormatG(raw*scale+offset, 6)
	if unit := text(point, "unit"); unit != "" {
		value += " " + unit
	}
	return value
}
