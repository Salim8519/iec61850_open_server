package netcfg

import (
	"regexp"
	"strings"

	"studio-engine/internal/pyjson"
)

// SNMPRulesScript lists the SNMP device firewall rules ScadaBench made (ADR-0092 amendment).
const SNMPRulesScript = "Get-NetFirewallRule -Name '" + SNMPRulePrefix + "*' -ErrorAction SilentlyContinue | " +
	"Select-Object -ExpandProperty Name"

var snmpRulePattern = regexp.MustCompile(`^` + regexp.QuoteMeta(SNMPRulePrefix) + `[0-9a-f]{12}$`)

// ListSNMPRules returns the names of the SNMP device rules present in Windows Firewall.
func ListSNMPRules(run Executor) ([]string, error) {
	text, err := shellQuery(run, SNMPRulesScript, adapterShellTimeout, adapterShellError)
	if err != nil {
		return nil, err
	}
	return nonEmptyLines(text), nil
}

// snmpDevices is the project's SNMP devices, none when Network setup has no SNMP side.
func (s *NetworkService) snmpDevices() ([]SNMPDevice, error) {
	if s.SNMP == nil {
		return nil, nil
	}
	return s.SNMP.SNMPDevices()
}

// splitRows separates the IED rows (Python's lab_ip) from the SNMP device rows.
func splitRows(rows []Row) (ieds []Row, snmp map[string]string) {
	snmp = map[string]string{}
	for _, row := range rows {
		if isSNMPRow(row) {
			snmp[strings.TrimPrefix(row.IEDID, SNMPRowPrefix)] = row.Proposed
		} else {
			ieds = append(ieds, row)
		}
	}
	return ieds, snmp
}

// validateSNMPFirewall: the helper only adds a ScadaBench SNMP rule for the SNMP worker beside
// the IED worker, on UDP ports and lab addresses.
func validateSNMPFirewall(item *pyjson.Object, program string) error {
	if !pyMatch(snmpRulePattern, textOr(item, "name")) || get(item, "group") != Group {
		return refuse("SNMP firewall item is not a ScadaBench rule")
	}
	ports, ok := get(item, "ports").([]any)
	if !ok || len(ports) == 0 {
		return refuse("SNMP firewall ports are invalid")
	}
	for _, port := range ports {
		if !pyIntIn(port, 1, 65535) {
			return refuse("SNMP firewall ports are invalid")
		}
	}
	addresses, ok := get(item, "addresses").([]any)
	if !ok || len(addresses) == 0 {
		return refuse("SNMP firewall rule needs at least one address")
	}
	for _, address := range addresses {
		if err := labAddress(address); err != nil {
			return err
		}
	}
	if !samePath(textOr(item, "program"), snmpWorker(program)) {
		return refuse("SNMP firewall rule must name the ScadaBench SNMP engine")
	}
	return nil
}

// snmpFirewallCommand is the PowerShell that adds the SNMP device rule.
func snmpFirewallCommand(f *itemFields) string {
	return "New-NetFirewallRule -Name " + psString(f.text("name")) + " -DisplayName " +
		psString("ScadaBench simulated SNMP devices") + " -Group " + psString(f.text("group")) + " " +
		"-Direction Inbound -Action Allow -Protocol UDP -LocalPort " + f.joined("ports", ",") + " " +
		"-LocalAddress " + f.joined("addresses", ",") + " -Program " + psString(f.text("program")) + " -Profile Any"
}
