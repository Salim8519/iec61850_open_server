package netcfg

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/licensing/hostkey"
	"studio-engine/internal/pyjson"
)

// The Windows Firewall rule of the VM host key (ADR-0099). The host helper listens on TCP port
// 47862 only on the adapters of virtual switches; this one allow rule lets the virtual machines
// reach it. It goes through the same previewed, journaled, removable flow as every other change
// ScadaBench makes: an exact preview with a fingerprint, the elevated helper, the journal, and
// Settings > Network > Clean up to remove it again.
const (
	// HostKeyPrefix names the one rule, followed by 12 hex digits of the helper program's path hash.
	HostKeyPrefix = "ScadaBench-VMHost-"
	// HostKeyKind is the item kind in plans, the journal and the helper's allow-list.
	HostKeyKind = "hostkey_firewall"
)

var hostKeyRulePattern = regexp.MustCompile(`^` + regexp.QuoteMeta(HostKeyPrefix) + `[0-9a-f]{12}$`)

// hostKeyWorker is the host helper beside the IED worker.
func hostKeyWorker(program string) string {
	return filepath.Join(filepath.Dir(program), hostkey.HelperExe)
}

// HostKeyRuleName is HostKeyPrefix + sha256(lower(program))[:12].
func HostKeyRuleName(program string) string {
	return HostKeyPrefix + sha256Hex([]byte(strings.ToLower(program)))[:12]
}

// HostKeyPlan is the one allow rule for the host helper, unless it already exists.
func HostKeyPlan(program string, existing []string) (Plan, error) {
	name := HostKeyRuleName(program)
	present := contains(existing, name)
	item := pyjson.NewObject().Set("kind", HostKeyKind).Set("name", name).Set("group", GatewayGroup).
		Set("ports", intsAny([]int{hostkey.Port})).Set("program", program).Set("owned", !present).
		Set("action", keepOrAdd(present))
	fingerprint, err := sortedFingerprint(pyjson.NewObject().Set("hostkey", []any{item}))
	return Plan{Items: []*pyjson.Object{item}, Fingerprint: fingerprint}, err
}

// validateHostKeyFirewall: the elevated helper only adds this rule for the host helper program,
// for TCP port 47862 and nothing else.
func validateHostKeyFirewall(item *pyjson.Object, program string) error {
	if !pyMatch(hostKeyRulePattern, textOr(item, "name")) || get(item, "group") != GatewayGroup {
		return refuse("Virtual machine firewall item is not a ScadaBench rule")
	}
	ports, ok := get(item, "ports").([]any)
	if !ok || len(ports) != 1 || !pyIntIn(ports[0], hostkey.Port, hostkey.Port) {
		return refuse("Virtual machine firewall rule must name port " + strconv.Itoa(hostkey.Port) + " only")
	}
	if !samePath(textOr(item, "program"), hostKeyWorker(program)) {
		return refuse("Virtual machine firewall rule must name the ScadaBench host helper")
	}
	return nil
}

// hostKeyFirewallCommand is the PowerShell that adds the rule. LocalSubnet lets only computers of
// the same virtual network in; the helper itself also listens on no other adapter.
func hostKeyFirewallCommand(f *itemFields) string {
	return "New-NetFirewallRule -Name " + psString(f.text("name")) + " -DisplayName " +
		psString("ScadaBench virtual machine license") + " -Group " + psString(f.text("group")) + " " +
		"-Direction Inbound -Action Allow -Protocol TCP -LocalPort " + f.joined("ports", ",") + " " +
		"-RemoteAddress LocalSubnet -Program " + psString(f.text("program")) + " -Profile Any"
}

// ListHostKeyRules returns the names of the host helper rules present in Windows Firewall.
func ListHostKeyRules(run Executor) ([]string, error) {
	text, err := shellQuery(run, "Get-NetFirewallRule -Name '"+HostKeyPrefix+"*' -ErrorAction SilentlyContinue | "+
		"Select-Object -ExpandProperty Name", 20*time.Second, firewallReadError)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(text), nil
}

// HostKeyFirewall is the Settings flow of the rule: an exact, fingerprinted preview; nothing
// changes until the user confirms; the elevated helper applies it and the clean-up undoes it.
type HostKeyFirewall struct {
	Network HelperRunner
	Program string // path of scadabench_hostkey.exe
	Rules   func() ([]string, error)

	mu       sync.Mutex
	previews map[string]bool
}

// NewHostKeyFirewall uses the real read-only firewall query.
func NewHostKeyFirewall(network HelperRunner, program string, run Executor) *HostKeyFirewall {
	return &HostKeyFirewall{Network: network, Program: program, Rules: func() ([]string, error) {
		return RuleNames(HostKeyPrefix, func() ([]string, error) { return ListHostKeyRules(run) })
	}}
}

// Preview lists the one change the host helper needs.
func (f *HostKeyFirewall) Preview() (*pyjson.Object, error) {
	rules, err := f.Rules()
	if err != nil {
		return nil, checked(err)
	}
	plan, err := HostKeyPlan(f.Program, rules)
	if err != nil {
		return nil, err
	}
	items, err := withDescriptions(plan.Items, Describe)
	if err != nil {
		return nil, err
	}
	open := get(items[0], "action") == "keep"
	f.mu.Lock()
	if f.previews == nil {
		f.previews = map[string]bool{}
	}
	f.previews[plan.Fingerprint] = true
	f.mu.Unlock()
	return pyjson.NewObject().Set("items", itemsAny(items)).Set("fingerprint", plan.Fingerprint).
		Set("open", open).Set("needs_admin", !open), nil
}

// Apply makes the reviewed change after the user confirmed it.
func (f *HostKeyFirewall) Apply(fingerprint, confirm any) (*pyjson.Object, error) {
	if !isTrue(confirm) {
		return nil, apperr.New("Review the Windows Firewall change and confirm it first")
	}
	key, isText := fingerprint.(string)
	f.mu.Lock()
	known := isText && f.previews[key]
	f.mu.Unlock()
	if !known {
		return nil, conflict("This preview has expired. Review the change again.", nil)
	}
	fresh, err := f.Preview()
	if err != nil {
		return nil, err
	}
	if digest, _ := fresh.Get("fingerprint"); digest != key {
		return nil, conflict("Windows Firewall changed since you reviewed it. Review the new list.", digest)
	}
	raw, _ := fresh.Get("items")
	results := []any{}
	if adds := ownedAdds(objects(raw)); len(adds) > 0 {
		if results, err = f.Network.RunHelper("apply", adds, key); err != nil {
			return nil, err
		}
	}
	failed := false
	for _, entry := range results {
		failed = failed || !truthy(get(entry.(*pyjson.Object), "ok"))
	}
	message := "Done: Settings › Network › Clean up undoes this."
	if failed {
		message = "The change failed; see the line for the reason."
	}
	return pyjson.NewObject().Set("results", results).Set("applied", !failed).Set("message", message), nil
}

// HostKeyRuleNames is what the clean-up looks at to know the rule still exists.
func (s *NetworkService) HostKeyRuleNames() ([]string, error) {
	if s.HostKeyRules == nil {
		return nil, nil
	}
	return s.HostKeyRules()
}
