package netcfg

import (
	"errors"
	"sync"

	"studio-engine/internal/apperr"
	"studio-engine/internal/pyjson"
)

// HelperRunner runs owned changes through the elevated helper and journals them
// (NetworkService.RunHelper).
type HelperRunner interface {
	RunHelper(action string, items []*pyjson.Object, fingerprint string) ([]any, error)
}

// GatewayFirewall is gateway_firewall.GatewayFirewall (ADR-0062): an exact, fingerprinted
// preview; nothing changes until the user confirms; the helper applies it and the lab-network
// clean-up undoes it.
type GatewayFirewall struct {
	Network  HelperRunner
	Program  string // path of studio_gateway.exe
	Rules    func() ([]string, error)
	Blocking func(program string) ([]BlockingRule, error)

	mu       sync.Mutex
	previews map[string]bool
}

// NewGatewayFirewall uses the real read-only firewall queries.
func NewGatewayFirewall(network HelperRunner, program string, run Executor) *GatewayFirewall {
	return &GatewayFirewall{Network: network, Program: program,
		Rules:    func() ([]string, error) { return ListGatewayRules(run) },
		Blocking: func(program string) ([]BlockingRule, error) { return BlockingRules(program, run) }}
}

// checked wraps the errors Python catches (OSError, RuntimeError, ValueError).
func checked(err error) error {
	var known *apperr.Error
	var crash *pyCrash
	if errors.As(err, &crash) || (errors.As(err, &known) && known.Kind != apperr.Runtime && known.Kind != apperr.Value) {
		return err
	}
	return apperr.New("Windows Firewall could not be checked: " + err.Error())
}

// Preview lists the firewall changes the gateway needs.
func (g *GatewayFirewall) Preview() (*pyjson.Object, error) {
	rules, err := g.Rules()
	if err != nil {
		return nil, checked(err)
	}
	blocking, err := g.Blocking(g.Program)
	if err != nil {
		return nil, checked(err)
	}
	plan, err := GatewayPlan(g.Program, rules, blocking)
	if err != nil {
		return nil, err
	}
	items, err := withDescriptions(plan.Items, Describe)
	if err != nil {
		return nil, err
	}
	open, blocked := true, false
	for _, item := range items {
		open = open && get(item, "action") == "keep"
		blocked = blocked || get(item, "kind") == "unblock"
	}
	g.mu.Lock()
	if g.previews == nil {
		g.previews = map[string]bool{}
	}
	g.previews[plan.Fingerprint] = true
	g.mu.Unlock()
	return pyjson.NewObject().Set("items", itemsAny(items)).Set("fingerprint", plan.Fingerprint).
		Set("open", open).Set("needs_admin", !open).Set("blocked", blocked), nil
}

// Apply makes the reviewed changes after the user confirmed them.
func (g *GatewayFirewall) Apply(fingerprint, confirm any) (*pyjson.Object, error) {
	if !isTrue(confirm) {
		return nil, apperr.New("Review the Windows Firewall changes and confirm them first")
	}
	key, isText := fingerprint.(string)
	g.mu.Lock()
	known := isText && g.previews[key]
	g.mu.Unlock()
	if !known {
		return nil, conflict("This preview has expired. Review the changes again.", nil)
	}
	fresh, err := g.Preview()
	if err != nil {
		return nil, err
	}
	if digest, _ := fresh.Get("fingerprint"); digest != key {
		return nil, conflict("Windows Firewall changed since you reviewed it. Review the new list.", digest)
	}
	raw, _ := fresh.Get("items")
	results := []any{}
	if adds := ownedAdds(objects(raw)); len(adds) > 0 {
		if results, err = g.Network.RunHelper("apply", adds, key); err != nil {
			return nil, err
		}
	}
	failed := false
	for _, entry := range results {
		failed = failed || !truthy(get(entry.(*pyjson.Object), "ok"))
	}
	message := "Done: Settings › Network › Clean up undoes this."
	if failed {
		message = "Some changes failed; see each line for the reason."
	}
	return pyjson.NewObject().Set("results", results).Set("applied", !failed).Set("message", message), nil
}
