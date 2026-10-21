package app

import (
	"studio-engine/internal/api"
	"studio-engine/internal/api/hostkeyapi"
	"studio-engine/internal/licensing/hostkey"
	"studio-engine/internal/netcfg"
)

// hostKeyRoutes wires the VM host key (ADR-0099): the Settings switch and the one previewed
// firewall rule. In a packaged app that has sharing switched on, the helper is also brought back
// (its Run value follows the app's folder, and it is started when it is not running).
func (e *Engine) hostKeyRoutes(network *netcfg.NetworkService) []api.Route {
	o := e.options
	helper := o.native(hostkey.HelperExe)
	sharing := hostkey.Sharing{Run: hostkey.SystemRunKey(), Process: hostkey.SystemProcess(), Executable: helper}
	firewall := netcfg.NewHostKeyFirewall(network, helper, netcfg.SystemExecutor{})
	if o.Enforce && !o.NoBackground {
		go func() {
			if err := sharing.Refresh(); err != nil {
				o.Logf("The virtual machine helper could not be restarted: %v", err)
			}
		}()
	}
	return hostkeyapi.New(hostkeyapi.Deps{License: e.License, Sharing: sharing, Firewall: firewall}).Routes()
}
