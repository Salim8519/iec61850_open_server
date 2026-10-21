package runtime

import "studio-engine/internal/pyjson"

// Bus publishes application events: it builds {"id", "kind", "timestamp", **payload} like
// runtime/event_bus.py and returns that envelope (stream A's SSE ring implements it).
type Bus interface {
	Publish(kind string, payload *pyjson.Object) *pyjson.Object
}

// Project is the runtime's view of the project actor (stream B). Current returns the
// immutable project tree with Python's keys and order; the mutations are the ones the
// Python coordinator made through ProjectService.
type Project interface {
	Current() *pyjson.Object
	// NodeData merges changes into a node's data and saves without project_changed
	// (ProjectService.node_data); errors are the project's StudioErrors.
	NodeData(nodeID string, changes *pyjson.Object) error
	// Source stores a validated source spec, saves and publishes project_changed.
	Source(iedID, ref string, spec *pyjson.Object) error
	// AssignIP stores a loopback address given at Run start to an IED without one
	// (networking/loopback.assign_endpoints mutated ied["ip"]; it is saved with the next save).
	AssignIP(iedID, ip string)
}

// Live is the acquisition side of a run (stream D): real-IED sessions, the GOOSE and SV
// listeners and the GOOSE engine. The Python coordinator called LiveManager at these points.
type Live interface {
	// Start runs at Run start with the project and the ids in the run.
	Start(project *pyjson.Object, active []string) error
	// Connect starts sessions for real IEDs added to a running run.
	Connect(project *pyjson.Object, real []*pyjson.Object) error
	Disconnect(iedID string)
	Stop()
	// BusFor registers publishing IEDs with the GOOSE engine: {ied_id: "udp:PORT:MAC"}.
	BusFor(ieds []*pyjson.Object) (map[string]string, error)
}

// NoLive is the Live used until stream D is merged: no real IEDs, no GOOSE engine.
type NoLive struct{}

func (NoLive) Start(*pyjson.Object, []string) error               { return nil }
func (NoLive) Connect(*pyjson.Object, []*pyjson.Object) error     { return nil }
func (NoLive) Disconnect(string)                                  {}
func (NoLive) Stop()                                              {}
func (NoLive) BusFor([]*pyjson.Object) (map[string]string, error) { return map[string]string{}, nil }
