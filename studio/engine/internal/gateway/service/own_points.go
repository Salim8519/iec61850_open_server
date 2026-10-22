package service

import (
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Own points (lane G, Go-only): values the user makes up. The page changes them at any
// time; while the gateway runs the change reaches SCADA at once (spontaneous / event).

// ownRow is the point table row of an own point.
func (m *Manager) ownRow(point *pyjson.Object) *pyjson.Object {
	row := copyObject(point).Set("ied_name", gateway.OwnName).Set("ied_role", "own").Set("monitor_only", false)
	kind := text(point, "kind")
	if strings.HasPrefix(kind, "command_") {
		return row.Set("value_text", commandText(kind, get(point, "value")))
	}
	current := m.own.current(text(point, "id"), get(point, "value"))
	return row.Set("value", current.Value).Set("value_text", Display(point, current)).Set("online", true)
}

// commandText says what the last command to an own command point was.
func commandText(kind string, value any) string {
	if value == nil {
		return "No command yet"
	}
	switch kind { // ADR-0107
	case gateway.KindStep:
		return map[bool]string{true: "Last command: raise", false: "Last command: lower"}[value == "higher"]
	case gateway.KindSetpoint:
		return "Last set point: " + pytext.Str(value)
	}
	number, _ := dictKey(value)
	if kind == "command_double" {
		return map[bool]string{true: "Last command: close", false: "Last command: open"}[number == 2]
	}
	return map[bool]string{true: "Last command: ON", false: "Last command: OFF"}[number == 1]
}

// savedPoint is the point with id in the saved list of the current protocol, or nil.
func (m *Manager) savedPoint(id string) *pyjson.Object {
	config := m.Config()
	for _, raw := range pointsOf(config, text(config, "protocol")) {
		if point, ok := raw.(*pyjson.Object); ok && get(point, "id") == id {
			return point
		}
	}
	return nil
}

// SetOwnValue changes the value of an own status or measured point: saved with the point
// list and, while the gateway runs, sent to SCADA at once. It answers the new table row.
func (m *Manager) SetOwnValue(id string, raw any) (*pyjson.Object, error) {
	point := m.savedPoint(id)
	if !gateway.IsOwn(point) {
		return nil, apperr.ValueError("This own point is not in the saved point list. Press Save point list first.")
	}
	kind := text(point, "kind")
	if gateway.IsCommandKind(kind) {
		return nil, apperr.ValueError("A command point has no value to type in; use its Send buttons.")
	}
	value, reason := gateway.OwnValue(kind, raw)
	if reason != "" {
		return nil, apperr.ValueError("The value was not changed: " + reason)
	}
	if err := m.applyOwn(map[string]any{id: value}); err != nil {
		return nil, err
	}
	if saved := m.savedPoint(id); saved != nil {
		point = saved
	}
	return m.ownRow(point), nil
}

// applyOwn sets own point values: first what the running gateway serves (SCADA sees it on
// the next 100 ms batch), then the saved point list, so a restart keeps them.
func (m *Manager) applyOwn(values map[string]any) error {
	now := time.Now().UnixMilli()
	var served []string
	for id, value := range values {
		if m.own.set(id, value, now) {
			served = append(served, id)
		}
	}
	m.mu.Lock()
	sync := m.sync
	m.mu.Unlock()
	if sync != nil && len(served) > 0 {
		sync.markOwn(served)
	}
	defer m.changed(ChangeGatewayPoints)
	m.own.persist.Lock()
	defer m.own.persist.Unlock()
	config := m.Config()
	protocol := text(config, "protocol")
	list := pointsOf(config, protocol)
	updated, found := make([]any, len(list)), false
	for index, raw := range list {
		updated[index] = raw
		if point, ok := raw.(*pyjson.Object); ok && gateway.IsOwn(point) {
			if value, hit := values[text(point, "id")]; hit {
				updated[index], found = with(point, "value", value), true
			}
		}
	}
	if !found {
		return nil
	}
	byProtocol, _ := get(config, "points").(*pyjson.Object)
	return m.projects.SaveGateway(with(config, "points", copyObject(byProtocol).Set(protocol, updated)))
}
