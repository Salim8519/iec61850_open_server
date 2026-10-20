package service

import (
	"encoding/hex"
	"fmt"
	"net/netip"
	"strings"

	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// MaxPoints is the most points one simulated device may have.
const MaxPoints = 100000

// CheckDevice returns the first problem of a device, worded for the user, or nil. others are
// the project's other devices (for the address check).
func CheckDevice(d model.Device, others []model.Device) error {
	name := strings.TrimSpace(d.Name)
	if name == "" || len(name) > 60 {
		return fmt.Errorf("Give the device a name of 1 to 60 characters")
	}
	address, err := netip.ParseAddr(d.IP)
	if err != nil || !address.Is4() {
		return fmt.Errorf("%s: %q is not an IPv4 address such as 127.60.0.1", name, d.IP)
	}
	if d.Port < 1 || d.Port > 65535 {
		return fmt.Errorf("%s: the port must be 1 to 65535 (SNMP uses 161)", name)
	}
	for _, other := range others {
		if other.ID != d.ID && other.IP == d.IP && other.Port == d.Port {
			return fmt.Errorf("%s: %s already uses %s port %d; give each device its own address", name, other.Name,
				d.IP, d.Port)
		}
	}
	if !d.V1 && !d.V2c && !d.V3 {
		return fmt.Errorf("%s: turn on at least one SNMP version", name)
	}
	if (d.V1 || d.V2c) && d.ReadCommunity == "" {
		return fmt.Errorf("%s: v1 and v2c need a read community, for example public", name)
	}
	if err := checkUsers(name, d); err != nil {
		return err
	}
	engine, err := hex.DecodeString(d.EngineID)
	if err != nil || len(engine) < 5 || len(engine) > 32 {
		return fmt.Errorf("%s: the engine ID must be 5 to 32 bytes written in hex", name)
	}
	if err := checkTraps(name, d); err != nil {
		return err
	}
	return checkPoints(name, d.Points)
}

func checkUsers(name string, d model.Device) error {
	if d.V3 && len(d.Users) == 0 {
		return fmt.Errorf("%s: add a v3 user, or turn v3 off", name)
	}
	seen := map[string]bool{}
	for _, user := range d.Users {
		if problem := proto.CheckUser(user); problem != "" {
			return fmt.Errorf("%s: %s", name, problem)
		}
		if seen[user.Name] {
			return fmt.Errorf("%s: user %s is listed twice", name, user.Name)
		}
		seen[user.Name] = true
	}
	return nil
}

func checkTraps(name string, d model.Device) error {
	users := map[string]bool{}
	for _, user := range d.Users {
		users[user.Name] = true
	}
	for _, target := range d.Traps {
		address, err := netip.ParseAddr(target.Host)
		if err != nil || !address.Is4() {
			return fmt.Errorf("%s: trap target %q is not an IPv4 address", name, target.Host)
		}
		if target.Port < 1 || target.Port > 65535 {
			return fmt.Errorf("%s: the trap port must be 1 to 65535 (usually 162)", name)
		}
		switch target.Version {
		case "v1", "v2c":
			if target.Inform && target.Version == "v1" {
				return fmt.Errorf("%s: SNMPv1 has no informs; choose v2c or v3 for %s", name, target.Host)
			}
			if target.Community == "" {
				return fmt.Errorf("%s: the trap target %s needs a community", name, target.Host)
			}
		case "v3":
			if !users[target.User] {
				return fmt.Errorf("%s: the trap target %s uses user %q, which the device does not have", name,
					target.Host, target.User)
			}
		default:
			return fmt.Errorf("%s: choose v1, v2c or v3 for the trap target %s", name, target.Host)
		}
	}
	return nil
}

func checkPoints(name string, points []model.Point) error {
	if len(points) > MaxPoints {
		return fmt.Errorf("%s has %d points; at most %d are allowed", name, len(points), MaxPoints)
	}
	seen := make(map[string]bool, len(points))
	for _, point := range points {
		oid, err := proto.ParseOID(point.OID)
		if err != nil {
			return fmt.Errorf("%s: %v", name, err)
		}
		key := oid.String()
		if seen[key] {
			return fmt.Errorf("%s: OID %s is listed twice", name, key)
		}
		seen[key] = true
		if _, err := proto.ParseValue(point.Type, point.Value, point.Hex); err != nil {
			return fmt.Errorf("%s: OID %s: %v", name, key, err)
		}
		switch point.Behaviour {
		case "", model.BehaviourFixed, model.BehaviourUptime, model.BehaviourCounter:
		default:
			return fmt.Errorf("%s: OID %s: unknown behaviour %q", name, key, point.Behaviour)
		}
		if point.Rate < 0 || point.Rate > 1e12 {
			return fmt.Errorf("%s: OID %s: the rate must be 0 to 1000000000000 per second", name, key)
		}
	}
	return nil
}
