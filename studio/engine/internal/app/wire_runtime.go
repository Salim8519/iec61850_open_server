package app

import (
	"errors"
	"sort"
	"strings"

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
		LabAddresses: e.labAddresses, LabRelease: e.labRelease, CommandKeep: o.CommandKeep, Version: o.Version})
	return live
}

// runtimeRoutes are api/runtime.py, equipment, scenarios runs, signals writes and
// GET /project/test-panel (runtimeapi owns the GET, toolsapi the PUT).
func (e *Engine) runtimeRoutes() []api.Route {
	return runtimeapi.Area{Runtime: e.Runtime, Project: e.Projects}.Routes()
}

// labAddresses puts the lab card's addresses back first when the project uses that card
// (ADR-0120: they are gone after Windows restarts), then lists the adapter's addresses.
func (e *Engine) labAddresses(guid string, starting []string) ([]string, error) {
	if e.Lab != nil {
		e.Lab.EnsureForRun(guid, starting)
	}
	return labAddresses(guid)
}

// labRelease takes the addresses of IEDs that stopped off the lab card (ADR-0127).
func (e *Engine) labRelease() {
	if e.Lab != nil {
		e.Lab.Release()
	}
}

// addressesOn is the IP Helper lookup (a variable so a test can make it fail).
var addressesOn = netcfg.AddressesOn

// labAddresses is ip_helper.addresses_on for the lab-mode preflight. A failed IP Helper
// call fails the Run as Python's OSError did: the error is no apperr (Flask's generic 500
// for the request) and its text is str(OSError(code, message)), "[Errno N] ...", in the
// runtime event.
func labAddresses(guid string) ([]string, error) {
	found, err := addressesOn(guid)
	if err != nil {
		return nil, osError(err)
	}
	addresses := make([]string, 0, len(found))
	for address := range found {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return addresses, nil
}

// osError rewrites netcfg's "[WinError N] text" as Python's str(OSError(N, text)).
func osError(err error) error {
	if rest, found := strings.CutPrefix(err.Error(), "[WinError "); found {
		return errors.New("[Errno " + rest)
	}
	return err
}
