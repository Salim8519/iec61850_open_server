package netcfg

import (
	"sort"
	"strings"

	"studio-engine/internal/pyjson"
)

// SNMP devices on the lab network (ADR-0092 amendment): they follow the IEDs onto the same
// adapter, each with its own address. Their rows use the IED planner (free addresses, ARP
// check, typed addresses); a project without SNMP devices plans exactly as before.

// SNMPRowPrefix marks a plan row (and a manual address key) that belongs to an SNMP device.
const SNMPRowPrefix = "snmp:"

// SNMPDevice is what Network setup reads of one simulated SNMP device.
type SNMPDevice struct {
	ID    string
	Name  string
	LabIP string // the address saved by an earlier Network setup ("" when none)
	Port  int
}

// SNMPProjects is the SNMP side of Network setup: the devices and where their addresses go.
type SNMPProjects interface {
	SNMPDevices() ([]SNMPDevice, error)
	// SetSNMPLabIPs saves each device's network address (device id -> address).
	SetSNMPLabIPs(addresses map[string]string) error
}

func isSNMPRow(row Row) bool { return strings.HasPrefix(row.IEDID, SNMPRowPrefix) }

// snmpManual is the typed addresses for SNMP rows plus each device's saved address when it
// is on this adapter's network and the user typed nothing for it.
func snmpManual(manual any, devices []SNMPDevice, adapter Adapter) (*pyjson.Object, map[string]bool, error) {
	result, saved := pyjson.NewObject(), map[string]bool{}
	networks, err := adapterNetworks(adapter)
	if err != nil {
		return nil, nil, err
	}
	for _, device := range devices {
		key := SNMPRowPrefix + device.ID
		override, err := manualOverride(manual, key)
		if err != nil {
			return nil, nil, err
		}
		if override != nil {
			result.Set(key, override)
			continue
		}
		ip, ok := parseIPv4(device.LabIP)
		if !ok {
			continue
		}
		for _, network := range networks {
			if network.Contains(ip) {
				result.Set(key, pyjson.NewObject().Set("ip", device.LabIP).Set("prefix", int64(network.Prefix)))
				saved[key] = true
				break
			}
		}
	}
	return result, saved, nil
}

// snmpRows plans one row per SNMP device: a free address on the adapter (never one an IED
// row uses), the saved address, or the typed one; each checked for other devices answering.
func (s *NetworkService) snmpRows(devices []SNMPDevice, adapter Adapter, manual any, iedRows []Row) ([]Row, error) {
	planned := make([]PlanIED, len(devices))
	for index, device := range devices {
		planned[index] = PlanIED{ID: SNMPRowPrefix + device.ID, Name: "SNMP · " + device.Name, Port: device.Port}
	}
	typed, saved, err := snmpManual(manual, devices, adapter)
	if err != nil {
		return nil, err
	}
	avoid := map[string]bool{}
	for _, row := range iedRows {
		if row.Proposed != "" {
			avoid[row.Proposed] = true
		}
	}
	var picked pickResult
	for round := 0; round < 2; round++ { // a saved address now used by another device is replaced
		if picked, err = s.pickAvoiding(planned, adapter, StrategyAdapter, typed, copySet(avoid)); err != nil {
			return nil, err
		}
		s.rowHints(picked)
		replaced := false
		for _, row := range picked.rows {
			if saved[row.IEDID] && row.Status != "ok" && row.Status != "already_on_adapter" {
				typed.Delete(row.IEDID)
				saved[row.IEDID], replaced = false, true
			}
		}
		if !replaced {
			break
		}
	}
	rows := picked.rows
	usedBy := map[string]string{}
	for _, row := range iedRows {
		if row.Proposed != "" {
			usedBy[row.Proposed] = row.Name
		}
	}
	for index := range rows {
		row := &rows[index]
		row.Kind, row.Saved = "snmp", saved[row.IEDID]
		if row.Saved && row.Status == "ok" {
			row.Hint = "Saved address of this device."
		}
		if other, clash := usedBy[row.Proposed]; clash && row.usable() {
			row.Status = "duplicate"
			row.Hint = other + " already uses " + row.Proposed + ". Each device needs its own address."
		} else if row.Proposed != "" {
			usedBy[row.Proposed] = row.Name
		}
	}
	return rows, s.snmpAlternatives(rows, planned, adapter, avoid, picked.inUse)
}

// snmpAlternatives offers a free address for every SNMP row that cannot be used.
func (s *NetworkService) snmpAlternatives(rows []Row, planned []PlanIED, adapter Adapter, avoid, inUse map[string]bool) error {
	keep, broken := copySet(avoid), false
	for address := range inUse {
		keep[address] = true
	}
	for _, row := range rows {
		if row.usable() {
			keep[row.Proposed] = true
		} else {
			broken = true
		}
	}
	if !broken {
		return nil
	}
	all, err := s.Adapters(false)
	if err != nil {
		return err
	}
	free, err := PlanAddresses(planned, adapter, all, StrategyAdapter, keep)
	if err != nil {
		return err
	}
	for index, row := range rows {
		other := free[index]
		if !row.usable() && other.Status == "ok" && other.Proposed != row.Proposed {
			rows[index].Alternative = &Alternative{IP: other.Proposed, Prefix: other.Prefix}
		}
	}
	return nil
}

func copySet(set map[string]bool) map[string]bool {
	copied := make(map[string]bool, len(set))
	for key, value := range set {
		copied[key] = value
	}
	return copied
}

// SNMPRulePrefix names the one firewall rule for the SNMP devices (UDP, their addresses only).
const SNMPRulePrefix = "ScadaBench-SNMP-"

// SNMPRuleName is SNMPRulePrefix + sha256 of the sorted addresses, ports and worker path.
func SNMPRuleName(addresses []string, ports []int, program string) (string, error) {
	name, err := MMSRuleName(addresses, ports, program)
	if err != nil {
		return "", err
	}
	return SNMPRulePrefix + strings.TrimPrefix(name, RulePrefix), nil
}

// snmpFirewall is the allow rule letting other PCs reach the SNMP devices on UDP.
func snmpFirewall(addresses []string, ports []int, program string, existing []string) (*pyjson.Object, error) {
	sortedPorts := uniqueSortedPorts(ports)
	name, err := SNMPRuleName(addresses, sortedPorts, program)
	if err != nil {
		return nil, err
	}
	present := contains(existing, name)
	return pyjson.NewObject().Set("kind", "snmp_firewall").Set("name", name).Set("group", Group).
		Set("addresses", stringsAny(sortedAddresses(addresses))).Set("ports", intsAny(sortedPorts)).
		Set("program", program).Set("owned", !present).Set("action", keepOrAdd(present)), nil
}

func sortedAddresses(addresses []string) []string {
	sorted := append([]string(nil), addresses...)
	sort.Strings(sorted)
	return sorted
}
