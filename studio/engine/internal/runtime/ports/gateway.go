// Package ports adapts the runtime coordinator to the interfaces other services declare
// where a method name or a slice element type differs (speed plan S1). *runtime.Coordinator
// satisfies projectapi.Runtime and toolsapi.Runtime directly; the gateway and acquisition
// get the small adapters below, so the composition needs no glue code of its own.
package ports

import (
	"studio-engine/internal/gateway/service"
	"studio-engine/internal/pyjson"
	"studio-engine/internal/runtime"
)

// Gateway is the coordinator as gateway/service.Runtime: the gateway's Signal is the
// coordinator.index record and its Subscribe takes service.Change.
func Gateway(c *runtime.Coordinator) service.Runtime { return gateway{c} }

type gateway struct{ c *runtime.Coordinator }

func (g gateway) RunState() string { return g.c.RunState() }

func (g gateway) Signal(iedID, ref string) (service.Signal, bool) {
	record, ok := g.c.Record(iedID, ref)
	return service.Signal(record), ok
}

func (g gateway) Endpoint(iedID string) (string, any, error) { return g.c.Endpoint(iedID) }

func (g gateway) Emit(kind string, payload *pyjson.Object) { g.c.Emit(kind, payload) }

func (g gateway) Subscribe(notify func(service.Change)) func() {
	return g.c.Subscribe(func(change runtime.Change) { notify(service.Change(change)) })
}
