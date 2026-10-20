package service

import (
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/snmp/agent"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/profiles"
	"studio-engine/internal/snmp/proto"
)

// trapTimeout covers informs to every target (3 tries of 1.5 s each, plus discovery).
const trapTimeout = 20 * time.Second

// Points are a device's points for the page: live values while it runs, else the stored
// ones, each with its MIB name and value meaning.
func (s *Service) Points(id string) ([]any, bool, error) {
	device, err := deviceOf(s.Config(), id)
	if err != nil {
		return nil, false, err
	}
	points, live := device.Points, false
	if s.deviceRunning(id) {
		answer, err := s.call("agent_points", map[string]any{"device": id}, 10*time.Second)
		if err == nil {
			var current struct {
				Points []model.Point `json:"points"`
			}
			if FromPy(answer, &current) == nil {
				points, live = current.Points, true
			}
		}
	}
	rows, err := ToPy(points)
	if err != nil {
		return nil, false, err
	}
	list, _ := rows.([]any)
	for _, item := range list {
		if row, isObject := item.(*pyjson.Object); isObject {
			if _, has := row.Get("writable"); !has {
				row.Set("writable", false) // a simulated point is writable only when the user said so
			}
		}
	}
	return s.MIBs.Decorate(list), live, nil
}

// SetValue changes one point: stored in the project, and sent to the running device.
func (s *Service) SetValue(id, oid, value string, isHex bool) (*pyjson.Object, error) {
	config := s.Config()
	index, err := config.findDevice(id)
	if err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	device := &config.Devices[index]
	at, err := pointIndex(device.Points, oid)
	if err != nil {
		return nil, err
	}
	point := device.Points[at]
	parsed, err := proto.ParseValue(point.Type, value, isHex)
	if err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	point.Type, point.Value, point.Hex = proto.ValueText(parsed)
	device.Points[at] = point
	if err := s.save(config); err != nil {
		return nil, err
	}
	if s.deviceRunning(id) {
		if _, err := s.call("agent_set", map[string]any{"device": id, "oid": oid, "value": point.Value,
			"hex": point.Hex}, 10*time.Second); err != nil {
			return nil, err
		}
	}
	s.changed(ChangeValues)
	return ToObject(point)
}

// Actions lists the events a device can trigger.
func (s *Service) Actions(id string) ([]profiles.Action, error) {
	device, err := deviceOf(s.Config(), id)
	if err != nil {
		return nil, err
	}
	return profiles.Actions(device), nil
}

// Trigger runs an event: its value changes first, then its trap or inform to every target.
func (s *Service) Trigger(id, action string) ([]agent.Result, error) {
	device, err := deviceOf(s.Config(), id)
	if err != nil {
		return nil, err
	}
	if !s.deviceRunning(id) {
		return nil, apperr.ValueError("Start the devices first: traps are sent by a running device")
	}
	if len(device.Traps) == 0 {
		return nil, apperr.ValueError(device.Name + " has no trap targets; add one under Traps")
	}
	event, err := profiles.Apply(device, action)
	if err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	for _, change := range event.Changes {
		if _, err := s.SetValue(id, change.OID, change.Value, change.Hex); err != nil {
			return nil, err
		}
	}
	answer, err := s.call("agent_trap", map[string]any{"device": id,
		"trap": agent.Trap{Notification: event.Notification, VarBinds: event.VarBinds}}, trapTimeout)
	if err != nil {
		return nil, err
	}
	var sent struct {
		Results []agent.Result `json:"results"`
	}
	if err := FromPy(answer, &sent); err != nil {
		return nil, err
	}
	for _, result := range sent.Results {
		s.Requests.Add(pyjson.NewObject().Set("device", id).Set("peer", result.Target).Set("version", "").
			Set("op", "trap").Set("count", int64(len(event.VarBinds))).Set("oid", event.Notification).
			Set("result", result.Text).Set("held", int64(0)).Set("name", s.MIBs.Name(event.Notification)))
	}
	return sent.Results, nil
}
