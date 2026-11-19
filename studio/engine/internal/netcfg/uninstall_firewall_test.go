package netcfg

import (
	"strings"
	"testing"

	"studio-engine/internal/pyjson"
)

func mmsRule(name, program string) *pyjson.Object {
	return pyjson.NewObject().Set("kind", "firewall").Set("name", RulePrefix+name).Set("group", Group).
		Set("ports", []any{102}).Set("addresses", []any{"192.168.230.21"}).Set("program", program).
		Set("owned", true).Set("action", "add")
}

// The uninstaller undoes the journaled firewall changes that still exist (two rules, one Windows
// Block rule switched off) and nothing else: not an address, not a rule already gone, not a rule
// that names another program (fix plan B-3).
func TestFirewallCleanupUndoesOnlyJournaledFirewallChanges(t *testing.T) {
	gateway := gatewayWorker(vectorProgram)
	block := pyjson.NewObject().Set("kind", "unblock").Set("name", "TCP Query User{1}studio_gateway.exe").
		Set("display", "studio_gateway").Set("program", gateway).Set("owned", true).Set("action", "add")
	journaled := []*pyjson.Object{mmsRule("aaaaaaaaaaaa", vectorProgram), mmsRule("bbbbbbbbbbbb", vectorProgram),
		block, mmsRule("cccccccccccc", vectorProgram), mmsRule("dddddddddddd", `C:\Other\studio_ied.exe`),
		aliasItem("192.168.230.21", "add")}
	journal := NewJournal(t.TempDir())
	for _, item := range journaled {
		if _, err := journal.Append("applied", item, "f", ""); err != nil {
			t.Fatal(err)
		}
	}
	present := []string{RulePrefix + "aaaaaaaaaaaa", RulePrefix + "bbbbbbbbbbbb", RulePrefix + "dddddddddddd"}
	service := &NetworkService{Directory: t.TempDir(), Program: vectorProgram, Journal: journal,
		Rules: func() ([]string, error) { return present, nil },
		Firewall: func([]string) (map[string]bool, map[string]bool, error) {
			return map[string]bool{}, map[string]bool{"TCP Query User{1}studio_gateway.exe": true}, nil
		}}
	scripts := []string{}
	fake := ExecutorFunc(func(command Command) (Output, error) {
		scripts = append(scripts, command.Argv[len(command.Argv)-1])
		return Output{0, batchMarker + "0 ok\n" + batchMarker + "1 ok\n" + batchMarker + "2 ok\n", ""}, nil
	})
	undone, problem, err := service.FirewallCleanup(fake)
	if err != nil || undone != 3 {
		t.Fatalf("undone %d problem %q err %v", undone, problem, err)
	}
	if !strings.Contains(problem, "must name the ScadaBench IED worker") {
		t.Fatalf("a foreign rule was not reported: %q", problem)
	}
	script := strings.Join(scripts, "\n")
	for _, want := range []string{"Remove-NetFirewallRule -Name 'SubstationStudio-MMS-aaaaaaaaaaaa'",
		"Remove-NetFirewallRule -Name 'SubstationStudio-MMS-bbbbbbbbbbbb'", "Enable-NetFirewallRule -Name 'TCP Query User{1}"} {
		if !strings.Contains(script, want) {
			t.Fatalf("script lacks %s: %s", want, script)
		}
	}
	for _, never := range []string{"cccccccccccc", "dddddddddddd", "NetIPAddress"} {
		if strings.Contains(script, never) {
			t.Fatalf("script touches %s: %s", never, script)
		}
	}
	left, _ := journal.Outstanding()
	kinds := []string{}
	for _, item := range left {
		kinds = append(kinds, textOr(item, "kind")+" "+textOr(item, "name")+textOr(item, "ip"))
	}
	want := "firewall SubstationStudio-MMS-cccccccccccc|firewall SubstationStudio-MMS-dddddddddddd|alias 192.168.230.21"
	if got := strings.Join(kinds, "|"); got != want {
		t.Fatalf("outstanding %s want %s", got, want)
	}
}
