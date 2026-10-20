package app

import (
	"os"

	"studio-engine/internal/api"
	"studio-engine/internal/api/platformapi"
	"studio-engine/internal/licensing"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/updates"
)

// wirePlatform builds the license service (one instance: the 402 gate and the /license
// routes share it), the update check that asks the same license server, the lab-network
// service and the gateway firewall, and returns their routes.
func (e *Engine) wirePlatform() ([]api.Route, error) {
	o := e.options
	home := o.LicenseHome
	if home == "" {
		home = licensing.DefaultHome()
	}
	e.License, e.Updates = o.LicenseService, o.UpdateService
	if e.License == nil {
		e.License = licensing.New(licensing.Options{Home: home, DataDir: o.DataDir, Enforce: o.Enforce,
			NativeFolder: o.NativeDir, Logf: o.Logf})
	}
	if e.Updates == nil {
		e.Updates = updates.New(updates.Options{Directory: o.DataDir, ServerURL: e.License.ServerURL,
			Current: o.Version, Logf: o.Logf})
	}
	executable := o.AppExecutable
	if executable == "" {
		found, err := os.Executable()
		if err != nil {
			return nil, err
		}
		executable = found
	}
	projectNetwork := platformapi.ProjectNetwork{Service: e.Projects}
	network := netcfg.NewNetworkService(projectNetwork, o.DataDir, o.native("studio_ied.exe"), executable)
	firewall := netcfg.NewGatewayFirewall(network, e.Gateway.Executable, netcfg.SystemExecutor{})
	return platformapi.New(platformapi.Deps{License: e.License, Updates: e.Updates, Network: network,
		Firewall: firewall, Projects: projectNetwork, Runtime: e.Runtime, AppVersion: o.Version}).Routes(), nil
}
