//go:build windows

package netcfg

import (
	"errors"
	"slices"
	"testing"
	"time"
)

// TestRuleNamesMatchPowerShell reads ScadaBench's rule names from the local rule store and
// with the PowerShell query the app used before (both read-only) and requires the same set.
func TestRuleNamesMatchPowerShell(t *testing.T) {
	shell := SystemExecutor{}
	for _, check := range []struct {
		prefix string
		slow   func() ([]string, error)
	}{
		{studioRulePrefix, func() ([]string, error) { return ListStudioRules(shell) }},
		{SNMPRulePrefix, func() ([]string, error) { return ListSNMPRules(shell) }},
		{gatewayRulePrefix, func() ([]string, error) { return ListGatewayRules(shell) }},
	} {
		started := time.Now()
		fast, err := RuleNames(check.prefix, func() ([]string, error) {
			t.Fatal("the rule store could not be read")
			return nil, nil
		})
		quick := time.Since(started)
		if err != nil {
			t.Fatal(err)
		}
		started = time.Now()
		slow, err := check.slow()
		if err != nil {
			t.Fatal(err)
		}
		slices.Sort(fast)
		slices.Sort(slow)
		if !slices.Equal(fast, slow) {
			t.Fatalf("%s: store %v, PowerShell %v", check.prefix, fast, slow)
		}
		t.Logf("%s: %d rule(s), store %v, PowerShell %v", check.prefix, len(fast), quick, time.Since(started))
	}
}

// TestRuleNamesFallBackToPowerShell: when the store cannot be read the old query answers.
func TestRuleNamesFallBackToPowerShell(t *testing.T) {
	saved := ruleRegistryNames
	defer func() { ruleRegistryNames = saved }()
	ruleRegistryNames = func() ([]string, error) { return nil, errors.New("store closed") }
	names, err := RuleNames("X-", func() ([]string, error) { return []string{"X-1"}, nil })
	if err != nil || !slices.Equal(names, []string{"X-1"}) {
		t.Fatalf("fallback %v %v", names, err)
	}
	ruleRegistryNames = func() ([]string, error) { return []string{"x-2", "Other", "X-3"}, nil }
	names, _ = RuleNames("X-", nil)
	if !slices.Equal(names, []string{"x-2", "X-3"}) {
		t.Fatalf("prefix match must ignore case like PowerShell: %v", names)
	}
}
