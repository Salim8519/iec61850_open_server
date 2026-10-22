package netcfg

import (
	"strconv"

	"studio-engine/internal/apperr"
)

// Address strategies: the SCD's own addresses, or free addresses on the adapter's network.
const (
	StrategySCD     = "scd"
	StrategyAdapter = "adapter"
)

// adapterNetworks is address_plan._adapter_networks: the adapter's non-link-local networks.
func adapterNetworks(adapter Adapter) ([]ipv4Iface, error) {
	networks := []ipv4Iface{}
	for _, item := range adapter.Addresses {
		text := item.IP + "/" + strconv.Itoa(item.Prefix)
		iface, ok := parseInterface(text)
		if !ok {
			return nil, apperr.ValueError(stringRepr(text) + " does not appear to be an IPv4 interface")
		}
		if !isLinkLocal(iface.IP) {
			networks = append(networks, iface)
		}
	}
	return networks, nil
}

// OnLink is address_plan.on_link: the address lies in a network the adapter is already on,
// so ARP can check it.
func OnLink(address string, adapter Adapter) (bool, error) {
	networks, err := adapterNetworks(adapter)
	if err != nil || len(networks) == 0 {
		return false, err
	}
	ip, ok := parseIPv4(address)
	if !ok {
		return false, apperr.ValueError(stringRepr(address) + " does not appear to be an IPv4 address")
	}
	for _, network := range networks {
		if network.Contains(ip) {
			return true, nil
		}
	}
	return false, nil
}

func scdInterface(ied PlanIED) (ipv4Iface, bool) {
	if ied.SCLIP == "" {
		return ipv4Iface{}, false
	}
	mask := ied.SCLSubnet
	if mask == "" {
		mask = "255.255.255.0"
	}
	return parseInterface(ied.SCLIP + "/" + mask)
}

type planState struct {
	own       map[string]bool
	elsewhere map[string]string
	taken     map[string]bool
	chosen    map[string]string
}

func newPlanState(adapter Adapter, all []Adapter, avoid map[string]bool) planState {
	state := planState{own: adapter.ips(), elsewhere: map[string]string{}, taken: map[string]bool{},
		chosen: map[string]string{}}
	for _, other := range all {
		if other.GUID != adapter.GUID {
			for _, item := range other.Addresses {
				state.elsewhere[item.IP] = other.Name
			}
		}
	}
	for ip := range state.own {
		state.taken[ip] = true
	}
	for ip := range state.elsewhere {
		state.taken[ip] = true
	}
	for ip := range avoid {
		state.taken[ip] = true
	}
	return state
}

// PlanAddresses is address_plan.plan_addresses: one row per IED. avoid holds addresses
// another device answered on, so a free address is never proposed twice.
func PlanAddresses(ieds []PlanIED, adapter Adapter, all []Adapter, strategy string, avoid map[string]bool) ([]Row, error) {
	if strategy != StrategySCD && strategy != StrategyAdapter {
		return nil, apperr.ValueError("Address strategy must be 'scd' or 'adapter'")
	}
	networks, err := adapterNetworks(adapter)
	if err != nil {
		return nil, err
	}
	state := newPlanState(adapter, all, avoid)
	rows := []Row{}
	for _, ied := range ieds {
		scd, hasSCD := scdInterface(ied)
		row := Row{IEDID: ied.ID, Name: ied.Name, Status: "ok"}
		if hasSCD {
			row.SCDIP = formatIPv4(scd.IP)
		}
		if strategy == StrategySCD {
			if !hasSCD {
				row.Status = "missing_scd_address"
				row.Hint = "This IED has no IP address in the SCD. Switch to 'free addresses on the " +
					"adapter' or type an address."
			} else {
				row.Proposed, row.Prefix = formatIPv4(scd.IP), intPointer(scd.Prefix)
			}
		} else {
			state.adapterRow(&row, scd, hasSCD, networks)
		}
		state.check(&row)
		state.claim(row)
		rows = append(rows, row)
	}
	return rows, nil
}

func (s planState) adapterRow(row *Row, scd ipv4Iface, hasSCD bool, networks []ipv4Iface) {
	var network *ipv4Iface
	for index := range networks {
		if hasSCD && networks[index].Contains(scd.IP) {
			network = &networks[index]
			break
		}
	}
	if network == nil && len(networks) > 0 {
		network = &networks[0]
	}
	switch {
	case network == nil:
		row.Status = "adapter_has_no_network"
		row.Hint = "This adapter has no usable IPv4 network yet. Use the SCD addresses instead."
	case hasSCD && network.Contains(scd.IP) && !s.taken[formatIPv4(scd.IP)]:
		row.Proposed, row.Prefix = formatIPv4(scd.IP), intPointer(network.Prefix)
	default:
		free := ""
		network.eachHost(func(index int, host uint32) bool {
			text := formatIPv4(host)
			_, used := s.chosen[text]
			if index >= 20 && !s.taken[text] && !used {
				free = text
				return false
			}
			return true
		})
		row.Proposed, row.Prefix = free, intPointer(network.Prefix)
		if free == "" {
			row.Status, row.Hint = "network_full", "No free address left in this network."
		}
	}
}

func (s planState) check(row *Row) {
	address := row.Proposed
	if address == "" || row.Status != "ok" {
		return
	}
	if s.own[address] {
		row.Status = "already_on_adapter"
		row.Hint = "This address is already on the adapter. ScadaBench will use it but will not remove it " +
			"later, because ScadaBench did not add it."
	} else if name, found := s.elsewhere[address]; found {
		row.Status = "used_by_other_adapter"
		row.Hint = address + " already belongs to '" + name + "'. Pick another address or disconnect that adapter."
	} else if name, found := s.chosen[address]; found {
		row.Status = "duplicate"
		row.Hint = name + " already uses " + address + ". Each IED needs its own address."
	}
}

// ValidateManual is address_plan.validate_manual for a typed address and prefix.
func ValidateManual(address, prefix any) (string, int, error) {
	iface, ok := parseInterface(pyStr(address) + "/" + pyStr(prefix))
	if !ok {
		return "", 0, apperr.ValueError("Enter an IPv4 address such as 192.168.1.21 and a prefix from 8 to 30")
	}
	if iface.Prefix < 8 || iface.Prefix > 30 || iface.IP == iface.Network() || iface.IP == iface.Broadcast() {
		return "", 0, apperr.ValueError("Use a host address inside a /8 to /30 network")
	}
	if isLoopback(iface.IP) || isMulticast(iface.IP) || isLinkLocal(iface.IP) {
		return "", 0, apperr.ValueError("Loopback, multicast and 169.254.x.x addresses cannot be used for the lab")
	}
	return formatIPv4(iface.IP), iface.Prefix, nil
}
