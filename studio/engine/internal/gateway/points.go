package gateway

import (
	"studio-engine/internal/pyjson"
	"studio-engine/internal/tools/pytext"
)

// Protocols the station gateway serves (gateway_points.PROTOCOLS).
var Protocols = []string{"iec104", "modbus", "dnp3"}

// MonitorKinds, CommandKinds and Kinds are the gateway point types in Python's order.
var (
	MonitorKinds = []string{"single", "double", "float", "integer"}
	CommandKinds = []string{"command_single", "command_double"}
	Kinds        = []string{"single", "double", "float", "integer", "command_single", "command_double"}
)

// MaxPoints is the largest point list a gateway holds.
const MaxPoints = 5000

// startAddress is the first address of each block per protocol (gateway_points.START).
var startAddress = map[string]map[string]int64{
	"iec104": {"single": 10001, "double": 20001, "float": 30001, "integer": 40001, "command": 50001},
	"modbus": {"single": 0, "double": 0, "float": 1000, "integer": 5000, "command": 0},
	"dnp3":   {"single": 0, "double": 0, "float": 0, "integer": 0, "command": 0},
}

// PointID is point_id: "<ied>:<ref>:<kind>".
func PointID(iedID, ref, kind string) string { return iedID + ":" + ref + ":" + kind }

// pointIDOf is point_id for any values (the f-string uses str()).
func pointIDOf(ied, ref, kind any) string {
	return pytext.Str(ied) + ":" + pytext.Str(ref) + ":" + pytext.Str(kind)
}

// IsCommandKind reports kind in COMMAND_KINDS.
func IsCommandKind(kind string) bool { return kind == "command_single" || kind == "command_double" }

// Cell is one (space, address) pair a point occupies.
type Cell struct {
	Space   string
	Address int64
}

// space is the address space a kind lives in; points sharing a space must not overlap.
func space(protocol, kind string) string {
	if protocol == "iec104" {
		return "ioa"
	}
	if IsCommandKind(kind) {
		return "command"
	}
	if protocol == "modbus" {
		if kind == "single" {
			return "discrete"
		}
		return "register"
	}
	if kind == "float" || kind == "integer" {
		return "analog"
	}
	return kind
}

// width is the number of Modbus registers per value (1 for other protocols).
func width(protocol, kind string) int64 {
	if protocol == "modbus" && (kind == "float" || kind == "integer") {
		return 2
	}
	return 1
}

// cells is occupied() for a kind and an integer address.
func cells(protocol, kind string, address int64) []Cell {
	name := space(protocol, kind)
	count := width(protocol, kind)
	result := make([]Cell, 0, count)
	for offset := int64(0); offset < count; offset++ {
		result = append(result, Cell{Space: name, Address: address + offset})
	}
	return result
}

// Occupied is occupied(protocol, point): the cells used by one point whose kind is a
// string and whose address is an integer (as validate_points and propose produce them).
func Occupied(protocol string, point *pyjson.Object) []Cell {
	kind, _ := get(point, "kind").(string)
	address, _ := pytext.Int64(get(point, "address"))
	return cells(protocol, kind, address)
}
