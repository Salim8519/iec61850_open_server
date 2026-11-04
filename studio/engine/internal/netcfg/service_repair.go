package netcfg

import (
	"os"
	"path/filepath"

	"studio-engine/internal/labnet"
	"studio-engine/internal/pyjson"
)

// ServiceRepairKind sets up the ScadaBench network service inside the address plan's one Windows
// prompt (ADR-0136): installed when this PC never had it, started again when another program
// switched it off, updated when it is too old. A real card's addresses then go on at Start and
// come off at Stop without a trip to another page. It runs scadabench_labnet.exe install, the
// same program the lab page's Create and Repair use. Clean-up never removes the service: it is
// not a leftover.
const ServiceRepairKind = "service_repair"

// labnetProgram is scadabench_labnet.exe beside the IED worker.
func labnetProgram(program string) string {
	return filepath.Join(filepath.Dir(program), "scadabench_labnet.exe")
}

// canRepairService says whether this installation has the program that sets the service up.
func canRepairService(program string) bool {
	_, err := os.Stat(labnetProgram(program))
	return err == nil
}

func serviceRepairItem(program string) *pyjson.Object {
	return pyjson.NewObject().Set("kind", ServiceRepairKind).Set("name", labnet.ServiceName).
		Set("program", labnetProgram(program)).Set("action", "add").Set("owned", true)
}

// serviceRepairCommand runs the program elevated (the helper already is) and passes its reason on.
// It is one script block, so the helper's "| Out-Null" applies to all of it.
func serviceRepairCommand(f *itemFields) string {
	return "& { $result = Join-Path $env:TEMP ('scadabench-labnet-' + [guid]::NewGuid().ToString('N') + '.json'); " +
		"& " + psString(f.text("program")) + " install --result $result; $code = $LASTEXITCODE; " +
		"$answer = if (Test-Path $result) { (Get-Content -Raw $result | ConvertFrom-Json).message } " +
		"else { 'The ScadaBench network service could not be set up.' }; " +
		"Remove-Item $result -ErrorAction SilentlyContinue; if ($code -ne 0) { throw $answer } }"
}

func validateServiceRepair(item *pyjson.Object, program string) error {
	if get(item, "name") != labnet.ServiceName || !samePath(textOr(item, "program"), labnetProgram(program)) {
		return refuse("Only ScadaBench's own network service program can be run")
	}
	return nil
}

const describeServiceRepair = "Set up the ScadaBench network service, which puts these addresses on at Start and " +
	"takes them off at Stop (it was missing, switched off or out of date)"

// withServiceRepair puts the set-up step first, before the approvals it reads, when the plan has
// addresses the service will put on.
func withServiceRepair(items []*pyjson.Object, program string) []*pyjson.Object {
	for _, item := range items {
		if get(item, "kind") == ApprovalKind {
			return append([]*pyjson.Object{serviceRepairItem(program)}, items...)
		}
	}
	return items
}
