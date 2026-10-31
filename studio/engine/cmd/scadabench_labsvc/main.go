// scadabench_labsvc is the ScadaBench lab network service program (ADR-0120, ADR-0131). The lab
// network program (scadabench_labnet.exe) copies it to Program Files and registers it as the
// service; Windows starts it with "service". It only adds and removes the simulated IEDs'
// addresses on the ScadaBench Lab card. It is a separate program so the installer never copies
// itself, which Kaspersky stopped as PDM:Trojan.Win32.Generic. It has no window.
package main

import (
	"os"

	"studio-engine/internal/labnet"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "service" {
		os.Exit(2)
	}
	if err := labnet.RunService(); err != nil {
		os.Exit(1)
	}
}
