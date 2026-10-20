package netcfg

import (
	"strings"

	"studio-engine/internal/pyjson"
)

// psString quotes a value as a single-quoted PowerShell string (” escapes ').
func psString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// blockingCheck: the helper re-checks, at the moment of the change, that the rule still blocks
// exactly the gateway worker; anything else is refused instead of being switched off.
func blockingCheck(f *itemFields) string {
	return "$rule = Get-NetFirewallRule -Name " + psString(f.text("name")) + "; " +
		"if ($rule.Action -ne 'Block' -or $rule.Direction -ne 'Inbound' -or " +
		"(($rule | Get-NetFirewallApplicationFilter).Program -ne " + psString(f.text("program")) + ")) " +
		"{ throw 'This rule is not a Windows block rule for the ScadaBench gateway; nothing was changed' }; "
}

// restoreCheck: clean-up only switches the rule back on, so its action does not matter
// (change 0089); it must still be the inbound rule for the gateway.
func restoreCheck(f *itemFields) string {
	return "$rule = Get-NetFirewallRule -Name " + psString(f.text("name")) + "; " +
		"if ($rule.Direction -ne 'Inbound' -or " +
		"(($rule | Get-NetFirewallApplicationFilter).Program -ne " + psString(f.text("program")) + ")) " +
		"{ throw 'This rule no longer belongs to the ScadaBench gateway; nothing was changed' }; "
}

// ApplyCommand is change_plan.apply_command: the exact PowerShell that makes one change.
func ApplyCommand(item *pyjson.Object) (string, error) {
	f := fieldsOf(item)
	var command string
	switch f.text("kind") {
	case "gateway_firewall":
		command = "New-NetFirewallRule -Name " + psString(f.text("name")) + " -DisplayName " +
			psString("ScadaBench station gateway") + " -Group " + psString(f.text("group")) + " -Direction Inbound " +
			"-Action Allow -Protocol TCP -Program " + psString(f.text("program")) + " -Profile Any"
	case "unblock":
		command = blockingCheck(f) + "Disable-NetFirewallRule -Name " + psString(f.text("name"))
	case "snmp_firewall":
		command = snmpFirewallCommand(f)
	case "alias":
		command = "New-NetIPAddress -InterfaceIndex " + f.integer("adapter_index") + " -IPAddress " + f.text("ip") +
			" -PrefixLength " + f.integer("prefix") + " -AddressFamily IPv4 -SkipAsSource $true " +
			"-PolicyStore ActiveStore"
	default:
		command = "New-NetFirewallRule -Name " + psString(f.text("name")) + " -DisplayName " +
			psString("ScadaBench simulated IEDs (MMS)") + " -Group " + psString(f.text("group")) + " " +
			"-Direction Inbound -Action Allow -Protocol TCP -LocalPort " + f.joined("ports", ",") + " " +
			"-LocalAddress " + f.joined("addresses", ",") + " -Program " + psString(f.text("program")) + " -Profile Any"
	}
	return command, f.err()
}

// RemoveCommand is change_plan.remove_command: the exact PowerShell that undoes one change.
func RemoveCommand(item *pyjson.Object) (string, error) {
	f := fieldsOf(item)
	var command string
	switch f.text("kind") {
	case "unblock":
		command = restoreCheck(f) + "Enable-NetFirewallRule -Name " + psString(f.text("name"))
	case "alias":
		command = "Remove-NetIPAddress -InterfaceIndex " + f.integer("adapter_index") + " -IPAddress " + f.text("ip") +
			" -PolicyStore ActiveStore -Confirm:$false"
	default:
		command = "Remove-NetFirewallRule -Name " + psString(f.text("name"))
	}
	return command, f.err()
}
