// Package service runs the station gateway (studio_gateway), the SCADA master
// (studio_master) and the protocol converter: gateway/manager.py, commands.py,
// converter.py, master_manager.py and modbus_probe.py, with one change of design that the
// user only notices as speed: values reach the gateway when they change (value hub
// subscription, coalesced to at most one batch per 100 ms like Python's 10 Hz loop)
// instead of a diff over every point ten times a second, and command confirmation waits
// on the hub instead of polling every 100 ms.
package service

import "studio-engine/internal/pyjson"

// Projects is what the gateway needs from the project service (stream B).
type Projects interface {
	// IEDs returns the project's IED models (immutable snapshot; never mutated here).
	IEDs() []*pyjson.Object
	// Gateway returns the stored gateway section, or nil when the project has none.
	Gateway() *pyjson.Object
	// SaveGateway is ProjectService.gateway(config): store, save, changed("gateway");
	// on a save failure the previous section stays and the error is returned.
	SaveGateway(config *pyjson.Object) error
}

// Signal is one record of the runtime's signal index (coordinator.index).
type Signal struct {
	Value        any    // nil when the record has no value
	QualityRef   string // "" when the signal has no quality attribute
	TimestampRef string
	LiveState    string // "" when absent, which counts as "fresh"
}

// Change says what changed in the runtime: one value (IEDID and Ref), every value and the
// state of one IED (Ref == ""), or the run state and everything (IEDID == "").
type Change struct {
	IEDID string
	Ref   string
}

// Runtime is what the gateway needs from the IED runtime and its value hub (stream C).
type Runtime interface {
	RunState() string
	Signal(iedID, ref string) (Signal, bool)
	// Endpoint is supervisor.endpoint: where a running simulated IED listens; an error
	// ("IED is not running", a ValueError) when it does not.
	Endpoint(iedID string) (host string, port any, err error)
	Emit(kind string, payload *pyjson.Object)
	// Subscribe calls notify after every change until cancel is called; notify must not block.
	Subscribe(notify func(Change)) (cancel func())
}

func get(o *pyjson.Object, key string) any {
	if o == nil {
		return nil
	}
	value, _ := o.Get(key)
	return value
}

func text(o *pyjson.Object, key string) string {
	value, _ := get(o, key).(string)
	return value
}

func copyObject(source *pyjson.Object) *pyjson.Object {
	result := pyjson.NewObject()
	if source == nil {
		return result
	}
	for _, key := range source.Keys() {
		result.Set(key, get(source, key))
	}
	return result
}

// with returns {**source, key: value} (an existing key keeps its place).
func with(source *pyjson.Object, key string, value any) *pyjson.Object {
	return copyObject(source).Set(key, value)
}
