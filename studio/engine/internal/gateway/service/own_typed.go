package service

import (
	"math"

	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// ownTyped carries out a command step on an own point (ADR-0107). No device exists: a
// select or cancel is confirmed, a step moves the own tap position it drives by one, and a
// set point sets the own value it drives. The command is stored as the point's value.
func (e *executor) ownTyped(point *pyjson.Object, phase Phase, value any, seq int64, label string) (bool, string, error) {
	m := e.manager
	if phase != PhaseExecute {
		return e.confirm(seq, label, "done: own point, "+string(phase)+" confirmed, no device is operated")
	}
	kind := text(point, "kind")
	if flag, isBool := value.(bool); isBool && !gateway.IsTypedKind(kind) {
		return e.ownCommand(point, flag, seq, label)
	}
	values := map[string]any{text(point, "id"): value}
	observed := ""
	if target := text(point, "target"); target != "" {
		status := m.Point(target)
		if next, ok := ownTarget(m, kind, status, value); ok {
			values[target] = next
			observed = ownTitle(status) + " is now " + pytext.Str(next)
		} else {
			observed = "the point it drives (" + target + ") is not in the point list"
		}
	}
	if err := m.applyOwn(values); err != nil {
		return e.refuse(seq, label, err.Error())
	}
	m.Commands.Update(seq, pyjson.NewObject().Set("applied", "done: own point, no device is operated").
		Set("observed", observed))
	m.emit("gateway_command", pyjson.NewObject().Set("point", label).Set("result", "own point").Set("ok", true))
	return true, "", nil
}

// ownTarget is the next value of the own point a step or set point drives.
func ownTarget(m *Manager, kind string, status *pyjson.Object, value any) (any, bool) {
	if !gateway.IsOwn(status) || gateway.IsCommandKind(text(status, "kind")) {
		return nil, false
	}
	target := text(status, "kind")
	number, _ := pytext.Float(value)
	if kind == gateway.KindStep {
		current, _ := pytext.Float(m.own.current(text(status, "id"), get(status, "value")).Value)
		number = current + map[bool]float64{true: 1, false: -1}[value == "higher"]
	}
	switch target {
	case gateway.KindStepPosition:
		return int64(math.Max(-64, math.Min(63, math.Round(number)))), true
	case "integer":
		return int64(math.Max(math.MinInt32, math.Min(math.MaxInt32, math.Round(number)))), true
	case "float":
		return number, true
	}
	return nil, false
}
