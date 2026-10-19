package ports

import (
	"studio-engine/internal/acquisition"
	"studio-engine/internal/native"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
)

// Acquisition holds the coordinator's acquisition ports; set them in acquisition.Deps:
// Runtime, Events (Emitter), Bus (Publisher), Values, Samples, Equipment, States and Start.
type Acquisition struct {
	Runtime   acquisition.Runtime
	Events    acquisition.Emitter
	Bus       acquisition.Publisher
	Values    acquisition.Values
	Samples   acquisition.Samples
	Equipment acquisition.Equipment
	States    acquisition.IEDStates
	Start     acquisition.StartWorker
}

// ForAcquisition returns the coordinator as acquisition sees it.
func ForAcquisition(c *runtime.Coordinator) Acquisition {
	return Acquisition{Runtime: c, Events: c, Bus: c, Values: values{c.Values()}, Samples: samples{c},
		Equipment: equipment{c}, States: states{c}, Start: StartWorker}
}

// StartWorker is acquisition.StartWorker on the native framed client (Job Object, ordered
// events, synthetic worker_failed / native_diagnostic, *native.TimeoutError on timeouts).
func StartWorker(executable string, onEvent func(event *pyjson.Object)) (acquisition.Worker, error) {
	worker, err := native.Start(executable, onEvent)
	if err != nil {
		return nil, err
	}
	return worker, nil
}

type values struct{ v runtime.Values }

func (a values) Has(iedID string) bool { return a.v.Has(iedID) }

func (a values) Get(iedID, ref string) (any, any, bool) { return a.v.Get(iedID, ref) }

func (a values) SetLive(iedID string, items []acquisition.RefValue) {
	a.v.SetLive(iedID, refValues(items))
}

func (a values) MarkStale(iedID string) { a.v.MarkStale(iedID) }

type samples struct{ c *runtime.Coordinator }

func (a samples) Running(iedID string) bool { return a.c.SampleRunning(iedID) }

func (a samples) Apply(iedID string, items []acquisition.RefValue, source string) error {
	return a.c.ApplySamples(iedID, refValues(items), source)
}

type equipment struct{ c *runtime.Coordinator }

func (a equipment) PublishChanged(keys []acquisition.PointKey) {
	converted := make([]runtime.PointKey, len(keys))
	for i, key := range keys {
		converted[i] = runtime.PointKey(key)
	}
	a.c.PublishChanged(converted)
}

type states struct{ c *runtime.Coordinator }

func (a states) SetState(iedID, state string) { a.c.SetIEDState(iedID, state) }

func refValues(items []acquisition.RefValue) []runtime.RefValue {
	converted := make([]runtime.RefValue, len(items))
	for i, item := range items {
		converted[i] = runtime.RefValue(item)
	}
	return converted
}
