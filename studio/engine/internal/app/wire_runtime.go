package app

import (
	"sort"

	"studio-engine/internal/api"
	"studio-engine/internal/api/runtimeapi"
	"studio-engine/internal/events"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/runtime"
)

// wireRuntime creates the coordinator over the project service (runtime.Project) and the
// SSE ring. Live acquisition is bound late: the returned adapter receives the manager once
// acquisition, which needs the coordinator's ports, exists.
func (e *Engine) wireRuntime(bus *events.Bus) *liveAdapter {
	o := e.options
	live := &liveAdapter{}
	e.Runtime = runtime.New(runtime.Options{NativeDir: o.NativeDir, DataDir: o.DataDir,
		Bus: runtimeBus{bus: bus, logf: o.Logf}, Project: e.Projects, Live: live,
		LabAddresses: labAddresses, CommandKeep: o.CommandKeep})
	return live
}

// runtimeRoutes are api/runtime.py, equipment, scenarios runs, signals writes and
// GET /project/test-panel (runtimeapi owns the GET, toolsapi the PUT).
func (e *Engine) runtimeRoutes() []api.Route {
	return runtimeapi.Area{Runtime: e.Runtime, Project: e.Projects}.Routes()
}

// labAddresses is ip_helper.addresses_on for the lab-mode preflight. Python raised the
// OSError of a failed IP Helper call; here a failure lists no address, so the preflight
// refuses with its "not set up on the adapter" text.
func labAddresses(guid string) []string {
	found, err := netcfg.AddressesOn(guid)
	if err != nil {
		return nil
	}
	addresses := make([]string, 0, len(found))
	for address := range found {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return addresses
}
