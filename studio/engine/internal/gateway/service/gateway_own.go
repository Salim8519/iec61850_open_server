package service

import (
	"studio-engine/internal/apperr"
)

// The gateway's own address on the ScadaBench Lab card (ADR-0125): typed once, checked like an IED
// address, kept in the gateway block as "own_addresses" (Go engine only) and offered again in the
// Network list. Only programs on this PC reach it.

// OwnName is how the lab network page and its journal name the gateway.
const OwnName = "Station gateway"

// LabAddresses is the lab network service as the gateway needs it (labsetup.Service).
type LabAddresses interface {
	Ready() bool
	AddOwn(address, owner string) (string, error)
	EnsureOwn(address, owner string) error
	RemoveOwn(address, owner string)
}

// ownAddresses is the stored list of addresses typed for the gateway.
func ownAddresses(config interface{ Get(string) (any, bool) }) []string {
	raw, _ := config.Get("own_addresses")
	list, _ := raw.([]any)
	addresses := []string{}
	for _, item := range list {
		if address, ok := item.(string); ok && address != "" {
			addresses = append(addresses, address)
		}
	}
	return addresses
}

func isOwn(config interface{ Get(string) (any, bool) }, address string) bool {
	for _, own := range ownAddresses(config) {
		if own == address && address != "" {
			return true
		}
	}
	return false
}

// AddOwnAddress checks a typed address, puts it on the lab card, keeps it in the list and makes it
// the gateway's network.
func (m *Manager) AddOwnAddress(address string) error {
	if m.State() == "running" {
		return apperr.ValueError("Stop the gateway before changing its network")
	}
	if m.Lab == nil || !m.Lab.Ready() {
		return apperr.ValueError("Create the lab network card first (Network > This PC lab network). " +
			"An own address lives on that card.")
	}
	parsed, err := m.Lab.AddOwn(address, OwnName)
	if err != nil {
		return err
	}
	config := m.Config()
	if !isOwn(config, parsed) {
		list := []any{}
		for _, own := range ownAddresses(config) {
			list = append(list, own)
		}
		if err := m.projects.SaveGateway(with(config, "own_addresses", append(list, parsed))); err != nil {
			return err
		}
	}
	return m.SaveNetwork(parsed, text(m.Config(), "allowed_scada"))
}

// leaveOwn takes the previous own address off the card when the gateway moves elsewhere.
func (m *Manager) leaveOwn(previous, next string) {
	if m.Lab != nil && previous != next && isOwn(m.Config(), previous) {
		m.Lab.RemoveOwn(previous, OwnName)
	}
}

// ensureOwn puts the chosen own address back on the card before the gateway listens there.
func (m *Manager) ensureOwn(address string) error {
	if m.Lab == nil {
		return apperr.ValueError("The lab network is not available, so " + address + " cannot be used. Choose another network.")
	}
	return m.Lab.EnsureOwn(address, OwnName)
}

// OwnAddresses are the card addresses the gateway uses now (for the lab network page's checks).
func (m *Manager) OwnAddresses() map[string]string {
	config := m.Config()
	used := map[string]string{}
	if address := text(config, "listen_address"); text(config, "listen") == "network" && isOwn(config, address) {
		used[address] = OwnName
	}
	return used
}
