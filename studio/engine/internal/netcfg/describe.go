package netcfg

import "studio-engine/internal/pyjson"

// Describe is change_plan.describe: one plain sentence a beginner can understand.
func Describe(item *pyjson.Object) (string, error) {
	f := fieldsOf(item)
	var text string
	switch f.text("kind") {
	case "gateway_firewall":
		if f.text("action") == "keep" {
			text = "The Windows Firewall rule for the ScadaBench gateway already exists"
		} else {
			text = "Allow other computers to reach the ScadaBench gateway (IEC 104, DNP3 or Modbus TCP) through Windows Firewall"
		}
	case "snmp_firewall":
		if f.text("action") == "keep" {
			text = "The Windows Firewall rule for the simulated SNMP devices already exists"
		} else {
			text = "Allow other devices to reach the simulated SNMP devices on UDP port " + f.joined("ports", ", ") +
				" through Windows Firewall"
		}
	case HostKeyKind:
		if f.text("action") == "keep" {
			text = "The Windows Firewall rule for virtual machine licensing already exists"
		} else {
			text = "Allow virtual machines on this PC to ask it for its ScadaBench license (TCP port " +
				f.joined("ports", ", ") + ", only from the same virtual network) through Windows Firewall"
		}
	case "unblock":
		text = "Switch off a Windows Firewall rule that blocks the ScadaBench gateway (Windows made it when its " +
			"'allow access?' question was closed); clean-up switches it on again"
	case "alias":
		if f.text("action") == "keep" {
			text = "Use address " + f.text("ip") + " that is already on '" + f.text("adapter_name") + "' for " + f.text("for")
		} else {
			text = "Add address " + f.text("ip") + "/" + f.text("prefix") + " to '" + f.text("adapter_name") +
				"' for " + f.text("for")
		}
	default:
		if f.text("action") == "keep" {
			text = "The Windows Firewall rule for ScadaBench already exists"
		} else {
			text = "Allow other devices to reach the simulated IEDs on TCP port " + f.joined("ports", ", ") +
				" through Windows Firewall"
		}
	}
	return text, f.err()
}

// DescribeRemoval is change_plan.describe_removal.
func DescribeRemoval(item *pyjson.Object) (string, error) {
	f := fieldsOf(item)
	var text string
	switch f.text("kind") {
	case "gateway_firewall":
		text = "Remove the Windows Firewall rule that ScadaBench added for its gateway"
	case "snmp_firewall":
		text = "Remove the Windows Firewall rule that ScadaBench added for the simulated SNMP devices"
	case HostKeyKind:
		text = "Remove the Windows Firewall rule that ScadaBench added for virtual machine licensing"
	case "unblock":
		text = "Switch the Windows Firewall rule that blocked the gateway on again, as it was before"
	case "alias":
		text = "Remove address " + f.text("ip") + " that ScadaBench added to '" + f.text("adapter_name") + "'"
	default:
		text = "Remove the Windows Firewall rule that ScadaBench added for the simulated IEDs"
	}
	return text, f.err()
}
