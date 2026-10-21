// Package acquisition runs the live side of a Run: real-IED sessions through studio_live,
// live values, the GOOSE monitor/engine/inputs/sharing, the Sampled Values monitor, the MMS
// client list, station health and test-mode application (Go port of
// studio_app/acquisition and the related simulation helpers; stream D, ADR-0076).
//
// The package depends on the rest of the engine only through the small interfaces below;
// the integrating session wires them to stream B (project actor), stream C (runtime values,
// sample writer, native framing) and stream A (event bus).
package acquisition

import (
	"time"

	"studio-engine/internal/pyjson"
)

// Projects is the project actor as acquisition sees it. Current returns the immutable
// published project tree (never mutated after publication, shared by pointer).
type Projects interface {
	Current() *pyjson.Object
	// FindIED returns the IED or apperr Coded("Unknown IED", "not_found").
	FindIED(id string) (*pyjson.Object, error)
	// RunID is f"{session_id}-{revision}" of the health report.
	RunID() string
	// SetTestMode stores validated test settings on a simulated IED, saves and announces the
	// change; it returns the IED as published after the change.
	SetTestMode(iedID string, settings *pyjson.Object) (*pyjson.Object, error)
	// SaveReach is projects/goose_reach.save_reach: validate_network, save, changed; it
	// returns the StudioError of validate_network unchanged. reach is the request value
	// as sent (validate_network rejects anything but "inside"/"outside").
	SaveReach(reach any, uplinkGUID, shareGUID string) error
}

// Runtime is the coordinator state acquisition reads.
type Runtime interface {
	// State is "stopped", "running" or "paused".
	State() string
}

// Emitter is coordinator.emit: the event bus plus the run log while a run is open.
type Emitter interface {
	Emit(kind string, payload *pyjson.Object)
}

// Publisher is a bus-only publish (never in the run log), as mms_access uses.
type Publisher interface {
	Publish(kind string, payload *pyjson.Object)
}

// Values is the runtime value table of the IEDs in the run (coordinator.ied_indexes).
type Values interface {
	// Has reports whether the IED has a runtime index (it is part of the run).
	Has(iedID string) bool
	// Get returns a point's current value and live_state (nil when never set).
	Get(iedID, ref string) (value any, liveState any, ok bool)
	// SetLive stores device values and live_state "fresh" under one lock.
	SetLive(iedID string, values []RefValue)
	// MarkStale turns every "fresh" live_state of the IED into "stale".
	MarkStale(iedID string)
}

// RefValue is one point value.
type RefValue struct {
	Ref   string
	Value any
}

// Samples is the coordinator's sample writer (writes into simulated IED workers).
type Samples interface {
	// Running is `ied_id in sample_writer.locks`.
	Running(iedID string) bool
	// Apply writes (ref, value) pairs with quality None; the error text is Python's ValueError.
	Apply(iedID string, values []RefValue, source string) error
}

// Equipment is coordinator.equipment_states.publish_changed.
type Equipment interface {
	PublishChanged(keys []PointKey)
}

// PointKey is an (IED id, reference) pair.
type PointKey struct {
	IED string
	Ref string
}

// IEDStates keeps the runtime-only `state` field of an IED ("real_live", "stopped", ...).
type IEDStates interface {
	SetState(iedID, state string)
}

// Worker is one framed native process (studio_live.exe or studio_goose.exe). Request
// returns the reply's "result", an apperr Runtime error for an ok:false reply (Python
// RuntimeError text) or a timeout error with text "Native <op> request timed out".
type Worker interface {
	Request(op string, timeout time.Duration, args *pyjson.Object) (any, error)
	Close()
}

// IsTimeout reports whether err is a request timeout (Python TimeoutError): such errors
// implement Timeout() bool, like net.Error.
func IsTimeout(err error) bool {
	timeout, ok := err.(interface{ Timeout() bool })
	return ok && timeout.Timeout()
}

// StartWorker starts executable and delivers every event frame (and the synthetic
// worker_failed / native_diagnostic events) to onEvent, in pipe order.
type StartWorker func(executable string, onEvent func(event *pyjson.Object)) (Worker, error)

// Network gives the adapters and the Npcap state (networking/adapters, networking/npcap).
type Network interface {
	// Adapters returns the normalized adapter dicts; an error stands for OSError/RuntimeError.
	Adapters() ([]*pyjson.Object, error)
	// Npcap returns npcap_status(): {installed, path, download, message}.
	Npcap() *pyjson.Object
}

// Clock gives wall time (timestamps) and monotonic seconds (ages, deadlines).
type Clock interface {
	Now() time.Time
	Monotonic() float64
}

// SubscriptionMembers completes a GOOSE subscription that lacks dataset positions by
// reading the stored SCL (acquisition/goose_inputs._with_members; stream B parses SCL).
type SubscriptionMembers func(ied, subscription *pyjson.Object) (*pyjson.Object, error)
