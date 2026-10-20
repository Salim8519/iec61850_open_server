package service

import (
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// ownCommand carries out a command (from SCADA or the page) on an own command point. No
// device exists, so it always succeeds: the command is stored as the point's value and the
// own status point it drives ("target") follows, which SCADA then sees as a status change.
// The Commands tab logs it like any other command.
func (e *executor) ownCommand(point *pyjson.Object, value bool, seq int64, label string) (bool, string, error) {
	m := e.manager
	kind := text(point, "kind")
	if !gateway.IsCommandKind(kind) {
		reason := "this point is not a command"
		m.Commands.Update(seq, pyjson.NewObject().Set("applied", "refused: "+reason))
		m.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("result", "refused").Set("reason", reason))
		return false, reason, nil
	}
	values := map[string]any{text(point, "id"): commandValue(kind, value)}
	observed := ""
	if target := text(point, "target"); target != "" {
		status := m.Point(target)
		if gateway.IsOwn(status) && !gateway.IsCommandKind(text(status, "kind")) {
			next := commandValue(map[bool]string{true: "command_double", false: "command_single"}[text(status, "kind") == "double"], value)
			values[target] = next
			observed = ownTitle(status) + " is now " + ownPosition(text(status, "kind"), next)
		} else {
			observed = "the status point it drives (" + target + ") is not in the point list"
		}
	}
	if err := m.applyOwn(values); err != nil {
		m.Commands.Update(seq, pyjson.NewObject().Set("applied", "refused: "+err.Error()))
		return false, err.Error(), nil
	}
	m.Commands.Update(seq, pyjson.NewObject().Set("applied", "done: own point, no device is operated").
		Set("observed", observed))
	m.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("result", "own point").Set("ok", true))
	return true, "", nil
}

// commandValue is the stored value of a command: 0/1 for single, 1 (open) / 2 (close) for double.
func commandValue(kind string, value bool) int64 {
	if kind == "command_double" {
		return map[bool]int64{true: 2, false: 1}[value]
	}
	return map[bool]int64{true: 1, false: 0}[value]
}

func ownPosition(kind string, value int64) string {
	if kind == "double" {
		return PositionText(value)
	}
	return map[bool]string{true: "ON", false: "OFF"}[value == 1]
}

// ownTitle names an own point in the Commands tab: its description, else its id.
func ownTitle(point *pyjson.Object) string {
	if name := pytext.Str(get(point, "description")); get(point, "description") != nil && name != "" {
		return name
	}
	return text(point, "id")
}
