package netcfg

import "strings"

// Fast, read-only listing of ScadaBench's own firewall rule names (change 0150). Windows keeps
// every rule made with New-NetFirewallRule in the local policy store, one registry value per
// rule named by the rule's Name, so reading the value names answers the same question
// "Get-NetFirewallRule -Name 'Prefix*'" answers, in milliseconds instead of seconds. When the
// store cannot be read the PowerShell query runs as before, so the answer never changes.

// ruleRegistryNames reads the value names of the local firewall rule store (registry_*.go).
var ruleRegistryNames = localRuleNames

// RuleNames lists the rules whose Name starts with prefix (case-insensitive, like PowerShell's
// wildcard), falling back to slow when the store cannot be read.
func RuleNames(prefix string, slow func() ([]string, error)) ([]string, error) {
	names, err := ruleRegistryNames()
	if err != nil {
		return slow()
	}
	wanted := strings.ToLower(prefix)
	found := []string{}
	for _, name := range names {
		if strings.HasPrefix(strings.ToLower(name), wanted) {
			found = append(found, name)
		}
	}
	return found, nil
}

// Rule name prefixes the PowerShell queries use.
const (
	studioRulePrefix  = "SubstationStudio-MMS-"
	gatewayRulePrefix = "ScadaBench-Gateway-"
)
