package netcfg

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"studio-engine/internal/pyjson"
)

// The Windows Firewall rule of the VM host key (ADR-0099): one allow rule for scadabench_hostkey.exe
// on TCP 47862, through the same preview, elevated helper, journal and clean-up as every other change.

const hostKeyProgram = `C:\app\native\scadabench_hostkey.exe`
const iedProgram = `C:\app\native\studio_ied.exe`

type hostKeyShell struct {
	mu       sync.Mutex
	commands []string
}

func (r *hostKeyShell) Run(command Command) (Output, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands = append(r.commands, command.Argv[len(command.Argv)-1])
	return Output{}, nil
}

// hostKeyRig is a NetworkService whose elevated helper really validates and runs the request
// (against a recording shell), and a rule list the test controls.
type hostKeyRig struct {
	t        *testing.T
	service  *NetworkService
	firewall *HostKeyFirewall
	shell    *hostKeyShell
	rules    []string
	launches int
}

func newHostKeyRig(t *testing.T) *hostKeyRig {
	t.Helper()
	rig := &hostKeyRig{t: t, shell: &hostKeyShell{}}
	ids := 0
	rig.service = NewNetworkService(&fakeProjects{mode: "loopback"}, t.TempDir(), iedProgram, `C:\app\ScadaBench.exe`)
	rig.service.NewID = func() string { ids++; return strings.Repeat("0", 31) + string(rune('0'+ids)) }
	rig.service.HostKeyRules = func() ([]string, error) { return rig.rules, nil }
	rig.service.Elevator = ElevatorFunc(func(executable string, arguments []string) (uint32, error) {
		rig.launches++
		if code := HelperMain(arguments[1:], rig.shell); code != 0 {
			t.Fatalf("the elevated helper exited with %d", code)
		}
		return 0, nil
	})
	rig.firewall = &HostKeyFirewall{Network: rig.service, Program: hostKeyProgram,
		Rules: func() ([]string, error) { return rig.rules, nil }}
	return rig
}

func firstItem(t *testing.T, preview *pyjson.Object) *pyjson.Object {
	t.Helper()
	raw, _ := preview.Get("items")
	items := objects(raw)
	if len(items) != 1 {
		t.Fatalf("the plan must hold exactly one change: %v", raw)
	}
	return items[0]
}

func TestThePlanIsOneAllowRuleForTheHostHelperOnItsPortAndNothingElse(t *testing.T) {
	rig := newHostKeyRig(t)
	preview, err := rig.firewall.Preview()
	if err != nil {
		t.Fatal(err)
	}
	item := firstItem(t, preview)
	if get(item, "kind") != "hostkey_firewall" || get(item, "owned") != true || get(item, "action") != "add" ||
		get(item, "group") != "ScadaBench" || get(item, "program") != hostKeyProgram ||
		!strings.HasPrefix(pyStr(get(item, "name")), "ScadaBench-VMHost-") || len(pyStr(get(item, "name"))) != len("ScadaBench-VMHost-")+12 {
		t.Fatalf("item %v", item)
	}
	if ports, _ := get(item, "ports").([]any); len(ports) != 1 || pyStr(ports[0]) != "47862" {
		t.Fatalf("ports %v", get(item, "ports"))
	}
	if get(preview, "open") != false || get(preview, "needs_admin") != true || pyStr(get(preview, "fingerprint")) == "" {
		t.Fatalf("preview %v", preview)
	}
	if text := pyStr(get(item, "description")); !strings.Contains(text, "virtual machines on this PC") ||
		!strings.Contains(text, "47862") || !strings.Contains(text, "only from the same virtual network") {
		t.Fatalf("description %q", text)
	}
}

func TestARuleThatAlreadyExistsIsKeptAndNeverOwned(t *testing.T) {
	rig := newHostKeyRig(t)
	rig.rules = []string{HostKeyRuleName(hostKeyProgram)}
	preview, _ := rig.firewall.Preview()
	item := firstItem(t, preview)
	if get(item, "action") != "keep" || get(item, "owned") != false || get(preview, "open") != true || get(preview, "needs_admin") != false {
		t.Fatalf("preview %v", preview)
	}
	if text := pyStr(get(item, "description")); text != "The Windows Firewall rule for virtual machine licensing already exists" {
		t.Fatalf("description %q", text)
	}
	result, err := rig.firewall.Apply(get(preview, "fingerprint"), true)
	if err != nil || rig.launches != 0 || get(result, "applied") != true {
		t.Fatalf("a rule that exists asked for administrator rights: %v launches %d", err, rig.launches)
	}
}

func TestApplyNeedsTheConfirmedFingerprintOfALivePreview(t *testing.T) {
	rig := newHostKeyRig(t)
	preview, _ := rig.firewall.Preview()
	fingerprint := get(preview, "fingerprint")
	if _, err := rig.firewall.Apply(fingerprint, nil); err == nil || !strings.Contains(err.Error(), "confirm") {
		t.Fatalf("without confirm: %v", err)
	}
	if _, err := rig.firewall.Apply("invented", true); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("an invented fingerprint: %v", err)
	}
	rig.rules = []string{HostKeyRuleName(hostKeyProgram)} // the firewall changed after the preview
	if _, err := rig.firewall.Apply(fingerprint, true); err == nil || !strings.Contains(err.Error(), "changed") {
		t.Fatalf("a firewall that changed since the preview: %v", err)
	}
	if rig.launches != 0 || len(rig.shell.commands) != 0 {
		t.Fatalf("something ran without a valid confirmation: %d launches", rig.launches)
	}
}

func TestApplyRunsTheExactCommandThroughTheHelperAndJournalsIt(t *testing.T) {
	rig := newHostKeyRig(t)
	preview, _ := rig.firewall.Preview()
	result, err := rig.firewall.Apply(get(preview, "fingerprint"), true)
	if err != nil || get(result, "applied") != true || rig.launches != 1 {
		t.Fatalf("apply: %v %v launches %d", result, err, rig.launches)
	}
	if len(rig.shell.commands) != 1 {
		t.Fatalf("commands %v", rig.shell.commands)
	}
	command := rig.shell.commands[0]
	name := HostKeyRuleName(hostKeyProgram)
	for _, part := range []string{"New-NetFirewallRule -Name '" + name + "'", "-Group 'ScadaBench'", "-Direction Inbound -Action Allow -Protocol TCP",
		"-LocalPort 47862", "-RemoteAddress LocalSubnet", "-Program '" + hostKeyProgram + "'", "-Profile Any"} {
		if !strings.Contains(command, part) {
			t.Errorf("the command misses %q: %s", part, command)
		}
	}
	for _, banned := range []string{"-RemoteAddress Any", "-LocalAddress", "-Action Block", "-Enabled False", "Disable-"} {
		if strings.Contains(command, banned) {
			t.Errorf("the command contains %q: %s", banned, command)
		}
	}
	records, _ := rig.service.Journal.Records()
	if len(records) != 1 || get(records[0], "event") != "applied" {
		t.Fatalf("journal %v", records)
	}
}

func TestCleanUpRemovesOnlyWhatTheJournalSaysWasAddedAndReportsTheLeftover(t *testing.T) {
	rig := newHostKeyRig(t)
	preview, _ := rig.firewall.Preview()
	rig.firewall.Apply(get(preview, "fingerprint"), true)
	rig.rules = []string{HostKeyRuleName(hostKeyProgram)} // Windows now has the rule
	leftovers, err := rig.service.Leftovers()
	if err != nil || len(leftovers) != 1 || get(leftovers[0], "kind") != "hostkey_firewall" {
		t.Fatalf("leftovers %v %v", leftovers, err)
	}
	cleanup, err := rig.service.CleanupPreview()
	if err != nil {
		t.Fatal(err)
	}
	if text := pyStr(get(firstItem(t, cleanup), "description")); text != "Remove the Windows Firewall rule that ScadaBench added for virtual machine licensing" {
		t.Fatalf("clean-up wording %q", text)
	}
	if _, err := rig.service.Cleanup(get(cleanup, "fingerprint"), true); err != nil {
		t.Fatal(err)
	}
	last := rig.shell.commands[len(rig.shell.commands)-1]
	if !strings.Contains(last, "Remove-NetFirewallRule -Name '"+HostKeyRuleName(hostKeyProgram)+"' | Out-Null") ||
		strings.Contains(last, "New-NetFirewallRule") {
		t.Fatalf("the removal command is %q", last)
	}
	rig.rules = nil
	if left, _ := rig.service.Leftovers(); len(left) != 0 {
		t.Fatalf("still leftover after clean-up: %v", left)
	}
	records, _ := rig.service.Journal.Records()
	if len(records) != 2 || get(records[1], "event") != "removed" {
		t.Fatalf("journal %v", records)
	}
}

func TestAPreExistingRuleIsNeverOwnedSoCleanUpNeverTouchesIt(t *testing.T) {
	rig := newHostKeyRig(t)
	rig.rules = []string{HostKeyRuleName(hostKeyProgram)}
	preview, _ := rig.firewall.Preview()
	rig.firewall.Apply(get(preview, "fingerprint"), true)
	if records, _ := rig.service.Journal.Records(); len(records) != 0 {
		t.Fatalf("a rule that was already there was journaled as ours: %v", records)
	}
	if cleanup, _ := rig.service.CleanupPreview(); len(objects(get(cleanup, "items"))) != 0 {
		t.Fatalf("clean-up would remove a rule ScadaBench did not add: %v", cleanup)
	}
}

func requestOf(t *testing.T, items ...*pyjson.Object) ([]byte, string) {
	t.Helper()
	list := make([]any, len(items))
	for index, item := range items {
		list[index] = item
	}
	payload, err := pyjson.Dumps(pyjson.NewObject().Set("action", "apply").Set("items", list))
	if err != nil {
		t.Fatal(err)
	}
	return payload, sha256Hex(payload)
}

func hostKeyItem() *pyjson.Object {
	plan, _ := HostKeyPlan(hostKeyProgram, nil)
	return copyObject(plan.Items[0])
}

func TestTheElevatedHelperRefusesAnythingButTheOneRuleForTheHostHelper(t *testing.T) {
	good := hostKeyItem()
	payload, digest := requestOf(t, good)
	if _, err := ValidateRequest(payload, digest, iedProgram); err != nil {
		t.Fatalf("the genuine request is refused: %v", err)
	}
	mutations := map[string]func(*pyjson.Object){
		"another port":        func(i *pyjson.Object) { i.Set("ports", []any{int64(80)}) },
		"two ports":           func(i *pyjson.Object) { i.Set("ports", []any{int64(47862), int64(445)}) },
		"no ports":            func(i *pyjson.Object) { i.Set("ports", []any{}) },
		"the port as text":    func(i *pyjson.Object) { i.Set("ports", []any{"47862; calc"}) },
		"another program":     func(i *pyjson.Object) { i.Set("program", `C:\Windows\System32\cmd.exe`) },
		"the IED worker":      func(i *pyjson.Object) { i.Set("program", iedProgram) },
		"another folder":      func(i *pyjson.Object) { i.Set("program", `C:\other\scadabench_hostkey.exe`) },
		"another name":        func(i *pyjson.Object) { i.Set("name", "ScadaBench-VMHost-zzzzzzzzzzzz") },
		"a name with a quote": func(i *pyjson.Object) { i.Set("name", "ScadaBench-VMHost-0123456789a';calc") },
		"a foreign rule name": func(i *pyjson.Object) { i.Set("name", "Remote Desktop - User Mode (TCP-In)") },
		"the old group":       func(i *pyjson.Object) { i.Set("group", "Substation Studio") },
		"a group of any kind": func(i *pyjson.Object) { i.Set("group", "Remote Desktop") },
		"not owned":           func(i *pyjson.Object) { i.Set("owned", false) },
		"an unknown kind":     func(i *pyjson.Object) { i.Set("kind", "hostkey_firewall2") },
		"no program at all":   func(i *pyjson.Object) { i.Delete("program") },
	}
	for name, change := range mutations {
		item := hostKeyItem()
		change(item)
		payload, digest := requestOf(t, item)
		if _, err := ValidateRequest(payload, digest, iedProgram); err == nil {
			t.Errorf("the helper accepted %s", name)
		}
	}
	if _, err := ValidateRequest(payload, "00"+digest[2:], iedProgram); err == nil {
		t.Error("the helper accepted a request that changed after it was prepared")
	}
	if _, err := ValidateRequest(payload, digest, filepath.Join(`C:\elsewhere`, "studio_ied.exe")); err == nil {
		t.Error("the helper accepted a request for programs of another install")
	}
}

func TestTheRuleNameFollowsTheProgramPathWithoutCase(t *testing.T) {
	if HostKeyRuleName(`C:\App\native\scadabench_hostkey.exe`) != HostKeyRuleName(`c:\app\NATIVE\SCADABENCH_HOSTKEY.EXE`) {
		t.Fatal("the same file under two spellings got two rules")
	}
	if HostKeyRuleName(hostKeyProgram) == HostKeyRuleName(`C:\other\scadabench_hostkey.exe`) {
		t.Fatal("two installs share one rule name")
	}
}
