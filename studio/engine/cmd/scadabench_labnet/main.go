// scadabench_labnet is the "This PC lab network" program (ADR-0120). ScadaBench starts it through
// the Windows administrator prompt to install ("install --result FILE": the ScadaBench Lab card,
// its settings, this program copied to Program Files and the ScadaBench lab network service) or
// to remove everything ("uninstall --result FILE"). Installed, Windows runs the copy as the
// service ("service"), which only adds and removes the simulated IEDs' addresses on that card.
// It has no window.
package main

import (
	"encoding/json"
	"fmt"
	"os"

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
	if len(args) == 1 && args[0] == "service" {
		if err := labnet.RunService(); err != nil {
			return 1
		}
		return 0
	}
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
		source, err := os.Executable()
		if err != nil {
			return write(args[2], outcome{Message: err.Error()})
		}
		installed, err := labnet.Install(source)
		if err != nil {
			return write(args[2], outcome{Message: err.Error(), Installed: &installed})
		}
		return write(args[2], outcome{OK: true, Message: "The lab network card is ready.", Installed: &installed})
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
