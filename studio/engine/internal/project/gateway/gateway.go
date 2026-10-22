package gateway

import (
	"studio-engine/internal/apperr"
	"studio-engine/internal/project/pyval"
	"studio-engine/internal/pyjson"
)

// Protocols are the station gateway protocols in their stored order.
var Protocols = []string{"iec104", "modbus", "dnp3"}

// Kinds are the gateway point kinds (monitor kinds, then command kinds).
var Kinds = []string{"single", "double", "float", "integer", "command_single", "command_double"}

// TypedKinds are the Go-only kinds of ADR-0107 (internal/gateway.TypedKinds): stored like the
// others; the frozen Python engine refuses them at Start.
var TypedKinds = []string{"step_position", "command_step", "command_setpoint"}

// MaxPoints is the largest point list per protocol.
const MaxPoints = 5000

// PointID is gateway_points.point_id: "<ied>:<ref>:<kind>" with Python str() of each part.
func PointID(iedID, ref, kind any) string {
	return pyval.Str(iedID) + ":" + pyval.Str(ref) + ":" + pyval.Str(kind)
}

// EmptyGateway is empty_gateway(): the gateway section of a new project.
func EmptyGateway() *pyjson.Object {
	settings := pyjson.NewObject()
	points := pyjson.NewObject()
	for _, name := range Protocols {
		settings.Set(name, defaults(gatewaySettings[name]))
		points.Set(name, []any{})
	}
	return pyjson.NewObject().Set("role", "gateway").Set("master", EmptyMaster()).Set("protocol", "iec104").
		Set("listen", "this_pc").Set("real_commands", true).Set("auto_start", false).
		Set("settings", settings).Set("points", points)
}

// ListenAddress is listen_address: the address the gateway listens on.
func ListenAddress(choice any) (string, error) {
	member, err := pyval.InStrings(choice, "this_pc", "network")
	if err != nil {
		return "", err
	}
	if !member {
		return "", apperr.ValueError("Choose 'This PC only' or 'All network adapters'")
	}
	if choice == "network" {
		return "0.0.0.0", nil
	}
	return "127.0.0.1", nil
}

// ValidateRole is master_rules.validate_role.
func ValidateRole(value any) (any, error) {
	if !pyval.InTuple(value, "gateway", "master", "converter") {
		return nil, apperr.ValueError("Choose Station gateway (gateway), SCADA master (master) or Protocol converter (converter)")
	}
	return value, nil
}

// NormalizeGateway is normalize_gateway: the stored gateway section checked and completed.
// Stored points are kept (a missing "id" is added to the stored point, as Python does);
// their IED references are checked only at Start.
func NormalizeGateway(raw any) (*pyjson.Object, error) {
	if raw == nil {
		return EmptyGateway(), nil
	}
	stored, isDict := raw.(*pyjson.Object)
	if !isDict {
		return nil, apperr.ValueError("The gateway section must be an object")
	}
	result := EmptyGateway()
	protocol := pyval.Lookup(stored, "protocol", "iec104")
	if !pyval.InTuple(protocol, Protocols...) {
		return nil, apperr.ValueError("Unknown gateway protocol '" + pyval.Str(protocol) + "'")
	}
	if _, err := ListenAddress(pyval.Lookup(stored, "listen", "this_pc")); err != nil {
		return nil, err
	}
	role, err := ValidateRole(pyval.Lookup(stored, "role", "gateway"))
	if err != nil {
		return nil, err
	}
	master, err := NormalizeMaster(pyval.Lookup(stored, "master", nil))
	if err != nil {
		return nil, err
	}
	result.Set("protocol", protocol).Set("listen", pyval.Lookup(stored, "listen", "this_pc")).
		Set("role", role).Set("master", master)
	for _, name := range []string{"real_commands", "auto_start"} {
		if value, ok := pyval.Lookup(stored, name, nil).(bool); ok {
			result.Set(name, value)
		}
	}
	settings, _ := result.Get("settings")
	points, _ := result.Get("points")
	for _, name := range Protocols {
		given, err := pyval.Get(pyval.Or(pyval.Lookup(stored, "settings", nil), pyjson.NewObject()), name, nil)
		if err != nil {
			return nil, err
		}
		checked, err := validateSettings(name, gatewaySettings[name], given, unsolicitedText)
		if err != nil {
			return nil, err
		}
		settings.(*pyjson.Object).Set(name, checked)
		list, err := storedPoints(stored, name)
		if err != nil {
			return nil, err
		}
		points.(*pyjson.Object).Set(name, list)
	}
	return result, nil
}

func unsolicitedText(string) string { return "Unsolicited reporting is on or off" }

func storedPoints(stored *pyjson.Object, name string) (any, error) {
	value, err := pyval.Get(pyval.Or(pyval.Lookup(stored, "points", nil), pyjson.NewObject()), name, []any{})
	if err != nil {
		return nil, err
	}
	list, isList := value.([]any)
	if !isList || len(list) > MaxPoints {
		return nil, apperr.ValueError("The " + name + " point list must be a list of at most 5000 points")
	}
	for _, item := range list {
		point, isDict := item.(*pyjson.Object)
		if !isDict || !pyval.InTuple(pyval.Lookup(point, "kind", nil), append(Kinds, TypedKinds...)...) ||
			!pyval.IsIntOrBool(pyval.Lookup(point, "address", nil)) {
			return nil, apperr.ValueError("A stored " + name + " gateway point has no valid type or address")
		}
		if !pyval.Has(point, "id") {
			kind, _ := point.Get("kind")
			point.Set("id", PointID(pyval.Lookup(point, "ied", ""), pyval.Lookup(point, "ref", ""), kind))
		}
	}
	return list, nil
}
