package netcfg

import (
	"reflect"
	"strings"
	"testing"
)

// Administrator prompts per Network setup Apply on a real card with the network service
// (ADR-0132), counted at the helper-request builder (trial N11, change 0225). Each Apply runs the
// elevated helper exactly once when ownedAdds is not empty, so one non-empty request = one
// Windows prompt. The bench does what Preview and the helper do to the plan: BuildPlan, approvals
// from the service's key, then the applied items become "approved" and "existing rule".

const benchProgram = `C:\ScadaBench\workers\studio_ied.exe`

type benchIED struct {
	id, ip string
	port   int
}

type promptBench struct {
	adapter  Adapter
	approved map[string]map[string]int
	rules    []string
	// firewallFree is the lower bound of every firewall idea: firewall items never need a prompt
	// (as if one rule already allowed everything, wider than today).
	firewallFree bool
	prompts      int
	requests     [][]string // item kinds of each elevated request
}

func newPromptBench(firewallFree bool) *promptBench {
	index := int64(7)
	return &promptBench{firewallFree: firewallFree, approved: map[string]map[string]int{},
		adapter: Adapter{GUID: "{11111111-2222-3333-4444-555555555555}", Name: "Ethernet", Index: &index,
			Kind: "physical", Addresses: []Address{{IP: "192.168.10.5", Prefix: 24}}}}
}

// apply is one Network setup Apply of these IEDs (each set to Simulate) on the card.
func (b *promptBench) apply(t *testing.T, ieds []benchIED) {
	t.Helper()
	rows, ports := []Row{}, []int{}
	for _, ied := range ieds {
		rows = append(rows, Row{IEDID: ied.id, Name: ied.id, Proposed: ied.ip, Prefix: intPointer(24), Status: "ok"})
		ports = append(ports, ied.port)
	}
	plan, err := BuildPlan(b.adapter, rows, ports, benchProgram, b.rules)
	if err != nil {
		t.Fatal(err)
	}
	adds := ownedAdds(useApprovals(plan.Items, b.approved))
	kinds := []string{}
	for _, item := range adds {
		f := fieldsOf(item)
		switch f.text("kind") {
		case ApprovalKind:
			guid := strings.ToUpper(f.text("adapter_guid"))
			if b.approved[guid] == nil {
				b.approved[guid] = map[string]int{}
			}
			b.approved[guid][f.text("ip")] = 24
		case "firewall":
			b.rules = append(b.rules, f.text("name"))
			if b.firewallFree {
				continue
			}
		}
		kinds = append(kinds, f.text("kind"))
	}
	if len(kinds) > 0 {
		b.prompts++
		b.requests = append(b.requests, kinds)
	}
}

// changes applies the first plan, then counts the prompts of the changes after it.
func (b *promptBench) changes(t *testing.T, plans [][]benchIED) int {
	t.Helper()
	b.apply(t, plans[0])
	before := b.prompts
	for _, plan := range plans[1:] {
		b.apply(t, plan)
	}
	return b.prompts - before
}

func threeIEDs(a string, port int, withC bool) []benchIED {
	plan := []benchIED{{"A", a, 102}, {"B", "192.168.10.31", port}}
	if withC {
		plan = append(plan, benchIED{"C", "192.168.10.32", 102})
	}
	return plan
}

// TestPromptsFiveNewAddresses is the owner's N11 check: one IED moved to a new address 5 times.
// Today: 5 prompts. With firewall items free (any firewall design): still 5, because each new
// address needs its approval written by the elevated helper (ADR-0132).
func TestPromptsFiveNewAddresses(t *testing.T) {
	plans := [][]benchIED{threeIEDs("192.168.10.20", 102, true)}
	for _, ip := range []string{"192.168.10.21", "192.168.10.22", "192.168.10.23", "192.168.10.24", "192.168.10.25"} {
		plans = append(plans, threeIEDs(ip, 102, true))
	}
	today, bound := newPromptBench(false), newPromptBench(true)
	if got := today.changes(t, plans); got != 5 {
		t.Fatalf("today: %d prompts for 5 new addresses, want 5", got)
	}
	if got := bound.changes(t, plans); got != 5 {
		t.Fatalf("firewall-free lower bound: %d prompts, want 5", got)
	}
	for _, request := range today.requests[1:] {
		if !reflect.DeepEqual(request, []string{ApprovalKind, "firewall"}) {
			t.Fatalf("an address change asks once for its approval and its rule together, got %v", request)
		}
	}
}

// TestPromptsBackAndForth: A moved between two addresses 5 times. Only the first visit of the
// second address asks; the approvals and the earlier rules stay until Clean up, so both count 1.
func TestPromptsBackAndForth(t *testing.T) {
	plans := [][]benchIED{threeIEDs("192.168.10.20", 102, true)}
	for round := 0; round < 5; round++ {
		ip := "192.168.10.21"
		if round%2 == 1 {
			ip = "192.168.10.20"
		}
		plans = append(plans, threeIEDs(ip, 102, true))
	}
	if got := newPromptBench(false).changes(t, plans); got != 1 {
		t.Fatalf("today: %d prompts, want 1", got)
	}
	if got := newPromptBench(true).changes(t, plans); got != 1 {
		t.Fatalf("firewall-free lower bound: %d prompts, want 1", got)
	}
}

// TestPromptsSimulateToggle: IED C switched off and on 5 times, every address already approved.
// Today the first smaller set asks once for its own rule; a firewall that never asks saves that
// one prompt. A rule that is not wider than today's must already allow the smaller set, so only
// reusing an existing, larger rule could save it.
func TestPromptsSimulateToggle(t *testing.T) {
	plans := [][]benchIED{threeIEDs("192.168.10.20", 102, true)}
	for round := 0; round < 5; round++ {
		plans = append(plans, threeIEDs("192.168.10.20", 102, round%2 == 1))
	}
	if got := newPromptBench(false).changes(t, plans); got != 1 {
		t.Fatalf("today: %d prompts, want 1", got)
	}
	if got := newPromptBench(true).changes(t, plans); got != 0 {
		t.Fatalf("firewall-free lower bound: %d prompts, want 0", got)
	}
}

// TestPromptsPortChange: IED B's MMS port changed 5 times, address unchanged. Today 5 prompts
// (one new rule each). Only a rule without a port filter, wider than today, would ask 0 times.
func TestPromptsPortChange(t *testing.T) {
	plans := [][]benchIED{threeIEDs("192.168.10.20", 102, true)}
	for _, port := range []int{10102, 10103, 10104, 10105, 10106} {
		plans = append(plans, threeIEDs("192.168.10.20", port, true))
	}
	if got := newPromptBench(false).changes(t, plans); got != 5 {
		t.Fatalf("today: %d prompts, want 5", got)
	}
	if got := newPromptBench(true).changes(t, plans); got != 0 {
		t.Fatalf("firewall-free lower bound: %d prompts, want 0", got)
	}
}
