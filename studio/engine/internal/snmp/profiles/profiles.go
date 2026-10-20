// Package profiles holds the ready-made simulated SNMP devices (ADR-0092): a network switch
// and a GPS time server, their points and the events the page can trigger on them.
package profiles

import (
	"fmt"

	"studio-engine/internal/snmp/model"
)

// Profile names stored in model.Device.Profile.
const (
	Switch = "switch"
	GPS    = "gps"
	Empty  = "empty"
	Walk   = "walk" // imported from snmpwalk output
	CSV    = "csv"  // imported from a ScadaBench CSV file
)

// Names are the page's words for each profile.
var Names = map[string]string{Switch: "Network switch", GPS: "GPS time server", Empty: "SNMP device",
	Walk: "Copied device", CSV: "Imported device"}

// SwitchPorts is how many ports a new switch has.
const SwitchPorts = 8

// Points returns the starting points of a profile (nil for an unknown one).
func Points(profile, name string) []model.Point {
	switch profile {
	case Switch:
		return switchPoints(name)
	case GPS:
		return gpsPoints(name)
	case Empty:
		return system(name, "ScadaBench simulated SNMP device", "0.0")
	}
	return nil
}

// system is the SNMPv2-MIB system group every profile starts with.
func system(name, description, objectID string) []model.Point {
	return []model.Point{
		text("1.3.6.1.2.1.1.1.0", description, false),
		{OID: "1.3.6.1.2.1.1.2.0", Type: model.TypeOID, Value: objectID},
		{OID: "1.3.6.1.2.1.1.3.0", Type: model.TypeTimeTicks, Value: "0", Behaviour: model.BehaviourUptime},
		text("1.3.6.1.2.1.1.4.0", "Substation engineer", true),
		text("1.3.6.1.2.1.1.5.0", name, true),
		text("1.3.6.1.2.1.1.6.0", "Substation control room", true),
		number("1.3.6.1.2.1.1.7.0", model.TypeInteger, 72),
	}
}

func text(oid, value string, writable bool) model.Point {
	return model.Point{OID: oid, Type: model.TypeOctets, Value: value, Writable: writable}
}

func number(oid, kind string, value int64) model.Point {
	return model.Point{OID: oid, Type: kind, Value: fmt.Sprint(value)}
}

func counter(oid, kind string, rate float64) model.Point {
	return model.Point{OID: oid, Type: kind, Value: "0", Behaviour: model.BehaviourCounter, Rate: rate}
}

// Action is one event the page can trigger on a running device.
type Action struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// Event is what an action does: the points it changes first, then the notification it sends.
type Event struct {
	Changes      []model.Point
	Notification string
	VarBinds     []model.Point
}

// coldStart and warmStart are the SNMPv2-MIB notifications every device can send.
const (
	coldStart = "1.3.6.1.6.3.1.1.5.1"
	warmStart = "1.3.6.1.6.3.1.1.5.2"
)

// Actions lists the events of a device: its profile's own, then cold and warm start.
func Actions(device model.Device) []Action {
	actions := []Action{}
	switch device.Profile {
	case Switch:
		actions = append(actions, switchActions(device)...)
	case GPS:
		actions = append(actions, gpsActions()...)
	}
	return append(actions, Action{ID: "cold_start", Label: "Restart (cold start trap)"},
		Action{ID: "warm_start", Label: "Warm start trap"})
}

// Apply turns an action into its event for device.
func Apply(device model.Device, action string) (Event, error) {
	switch action {
	case "cold_start":
		return Event{Notification: coldStart}, nil
	case "warm_start":
		return Event{Notification: warmStart}, nil
	}
	switch device.Profile {
	case Switch:
		if event, found := switchEvent(device, action); found {
			return event, nil
		}
	case GPS:
		if event, found := gpsEvent(action); found {
			return event, nil
		}
	}
	return Event{}, fmt.Errorf("unknown event %q for %s", action, device.Name)
}

// find is the point with oid, if the device has it.
func find(device model.Device, oid string) (model.Point, bool) {
	for _, point := range device.Points {
		if point.OID == oid {
			return point, true
		}
	}
	return model.Point{}, false
}
