package netcfg

import (
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

// The plan's set-up step runs only ScadaBench's own service program, first, and is never a
// leftover for clean-up (ADR-0136).
func TestServiceRepairStepRunsOnlyOurProgram(t *testing.T) {
	program := filepath.Join(t.TempDir(), "studio_ied.exe")
	approval := pyjson.NewObject().Set("kind", ApprovalKind).Set("action", "add")
	items := withServiceRepair([]*pyjson.Object{approval}, program)
	if len(items) != 2 || get(items[0], "kind") != ServiceRepairKind {
		t.Fatalf("items %v", items)
	}
	if err := validateItem(items[0], program); err != nil {
		t.Fatal(err)
	}
	command, err := ApplyCommand(items[0])
	if err != nil || !strings.Contains(command, "scadabench_labnet.exe' install --result $result") ||
		!strings.Contains(command, "throw $answer") {
		t.Fatalf("command %q %v", command, err)
	}
	if key, err := ItemKey(items[0]); err != nil || key != "service_repair:ScadaBenchLabNet" {
		t.Fatalf("key %q %v", key, err)
	}
	if text, err := Describe(items[0]); err != nil || !strings.Contains(text, "network service") {
		t.Fatalf("describe %q %v", text, err)
	}
	if present, err := (&presence{}).present(items[0]); present || err != nil {
		t.Fatalf("leftover %v %v", present, err)
	}

	other := copyObject(items[0]).Set("program", filepath.Join(t.TempDir(), "scadabench_labnet.exe"))
	if validateItem(other, program) == nil {
		t.Fatal("another folder's program accepted")
	}
	renamed := copyObject(items[0]).Set("name", "Spooler")
	if validateItem(renamed, program) == nil {
		t.Fatal("another service accepted")
	}
	firewall := pyjson.NewObject().Set("kind", "firewall").Set("action", "add")
	if got := withServiceRepair([]*pyjson.Object{firewall}, program); len(got) != 1 {
		t.Fatalf("no address, still a set-up step: %v", got)
	}
}
