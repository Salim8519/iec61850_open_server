package main

import (
	"fmt"
	"os"
	"path/filepath"

	"studio-engine/internal/netcfg"
)

// firewallCleanup undoes the firewall changes ScadaBench's network journal records as its own and
// still in place (fix plan B-3): its rules are removed and the Windows rules it switched off are
// switched back on. The journal is in the data folder; the IED worker beside this program is the
// one the rules must name.
func firewallCleanup(data string) outcome {
	self, err := os.Executable()
	if err != nil {
		return outcome{Message: err.Error()}
	}
	service := netcfg.NewNetworkService(nil, data, filepath.Join(filepath.Dir(self), "studio_ied.exe"), self)
	undone, problem, err := service.FirewallCleanup(netcfg.SystemExecutor{})
	switch {
	case err != nil:
		return outcome{Message: "ScadaBench could not read its record of firewall changes: " + err.Error()}
	case problem != "":
		return outcome{Message: fmt.Sprintf("%d firewall change(s) were undone, but not all: %s", undone, problem)}
	case undone == 0:
		return outcome{OK: true, Message: "ScadaBench had no firewall changes left to undo."}
	}
	return outcome{OK: true, Message: fmt.Sprintf("%d firewall change(s) ScadaBench made were undone.", undone)}
}
