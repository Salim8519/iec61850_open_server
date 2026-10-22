package service

import (
	"studio-engine/internal/gateway"
	"studio-engine/internal/pyjson"
)

// The start request of studio_gateway, extended by ADR-0107 and backward compatible both
// ways: a gateway program that answers hello with "typed_commands" gets
// "typed_commands": true and then hands every command step (select, execute, cancel) with
// what it decoded ("wire") to the engine. Without the flag, or with an older program, the
// outstation confirms selects and cancels itself, exactly as before.
const typedCapability = "typed_commands"

// nativePoint is a point as studio_gateway loads it; set points add their format.
func nativePoint(point *pyjson.Object, monitorOnly bool) *pyjson.Object {
	native := pyjson.NewObject().Set("id", get(point, "id")).Set("kind", get(point, "kind")).
		Set("address", get(point, "address")).Set("scale", get(point, "scale")).Set("monitor_only", monitorOnly)
	for _, name := range []string{"format", "word_order"} {
		if value, found := point.Get(name); found && get(point, "kind") == gateway.KindSetpoint {
			native.Set(name, value)
		}
	}
	return native
}

func hasTypedPoints(native []any) bool {
	for _, raw := range native {
		if point, ok := raw.(*pyjson.Object); ok && gateway.IsTypedKind(text(point, "kind")) {
			return true
		}
	}
	return false
}

func startRequest(protocol string, settings *pyjson.Object, native []any, typed bool) *pyjson.Object {
	request := pyjson.NewObject().Set("protocol", protocol).Set("settings", settings).Set("points", native)
	if typed {
		request.Set("typed_commands", true)
	}
	return request
}
