// scadabench_labnet is the "This PC lab network" program (ADR-0120). ScadaBench starts it through
// the Windows administrator prompt to install ("install --result FILE": the ScadaBench Lab card,
// its settings, the service program scadabench_labsvc.exe copied to Program Files and the
// ScadaBench lab network service) or to remove everything ("uninstall --result FILE"). The
// service is that separate program; this one never copies itself (ADR-0131). It has no window.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"studio-engine/internal/labnet"
)

// outcome is the result file the app reads after the prompt closes.
type outcome struct {
	OK        bool              `json:"ok"`
	Message   string            `json:"message"`
	Installed *labnet.Installed `json:"installed,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) (code int) {
	if len(args) != 3 || args[1] != "--result" || args[2] == "" {
		return 2
	}
	// Whatever happens, the app gets a result file with the reason (it has no other way to hear it).
	// The first write proves the program started and can write there.
	if write(args[2], outcome{Message: "The lab network program started but did not finish."}) == 1 &&
		!exists(args[2]) {
		return 3
	}
	defer func() {
		if failure := recover(); failure != nil {
			code = write(args[2], outcome{Message: fmt.Sprintf("The lab network program stopped unexpectedly: %v", failure)})
		}
	}()
	switch args[0] {
	case "install":
		self, err := os.Executable()
		if err != nil {
			return write(args[2], outcome{Message: err.Error()})
		}
		source := filepath.Join(filepath.Dir(self), labnet.ServiceProgram)
		if !exists(source) {
			return write(args[2], outcome{Message: "The lab network service program (" + labnet.ServiceProgram +
				") is missing from this installation. Reinstall ScadaBench."})
		}
		installed, err := labnet.Install(source)
		if err != nil {
			return write(args[2], outcome{Message: err.Error(), Installed: &installed})
		}
		ready := "The lab network card is ready."
		if installed.Note != "" {
			ready += " " + installed.Note // recorded in the network journal (ADR-0134)
		}
		return write(args[2], outcome{OK: true, Message: ready, Installed: &installed})
	case "start": // the service was set to manual or disabled by another program (ADR-0136)
		if err := labnet.TurnOn(); err != nil {
			return write(args[2], outcome{Message: err.Error()})
		}
		return write(args[2], outcome{OK: true, Message: "The ScadaBench network service is running again."})
	case "uninstall":
		if err := labnet.Uninstall(true); err != nil {
			return write(args[2], outcome{Message: err.Error()})
		}
		return write(args[2], outcome{OK: true, Message: "The lab network card and its service were removed."})
	}
	return 2
}

func write(path string, result outcome) int {
	data, _ := json.Marshal(result)
	if err := os.WriteFile(path, data, 0o666); err != nil {
		return 1
	}
	if !result.OK {
		return 1
	}
	return 0
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
