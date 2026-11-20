package service

import (
	"studio-engine/internal/apperr"
)

// The SCADA master's own address on the ScadaBench Lab card (ADR-0125, as the gateway's): typed
// once, checked like an IED address, kept in the master block as "own_addresses" (Go engine only)
// and used as the address the master connects from ("local"). Only devices on this PC answer it.
// It is on the card only while the master is connected: Connect puts it on, Disconnect and exit
// take it off (ADR-0141, trial N9).

// MasterOwnName is how the lab network page and its journal name the master.
const MasterOwnName = "SCADA master"

// AddOwnAddress checks a typed address, keeps it in the list and makes it the address the master
// connects from; it goes on the lab card at Connect.
func (m *Master) AddOwnAddress(address string) error {
	if m.Busy() {
		return apperr.ValueError("Disconnect before changing the network")
	}
	if m.Lab == nil || !m.Lab.Ready() {
		return apperr.ValueError("Create the lab network card first (Network > This PC lab network). " +
			"An own address lives on that card.")
	}
	parsed, err := m.Lab.CheckOwn(address, MasterOwnName)
	if err != nil {
		return err
	}
	config := m.Config()
	if !isOwn(config, parsed) {
		list := []any{}
		for _, own := range ownAddresses(config) {
			list = append(list, own)
		}
		if err := m.Save(with(config, "own_addresses", append(list, parsed))); err != nil {
			return err
		}
	}
	return m.SaveNetwork(parsed)
}

// ensureOwn holds the chosen own address and puts it on the card before the master connects from
// it (again before each reconnect: Windows forgets it at a restart).
func (m *Master) ensureOwn(config interface{ Get(string) (any, bool) }) error {
	local, _ := config.Get("local")
	address, _ := local.(string)
	if !isOwn(config, address) {
		return nil
	}
	if m.Lab == nil {
		return apperr.ValueError("The lab network is not available, so " + address + " cannot be used. Choose another network.")
	}
	return m.claim.take(m.Lab, address, MasterOwnName)
}

// leaveCard takes the own address off the card after a disconnect or a failed connect.
func (m *Master) leaveCard() { m.claim.give(m.Lab, MasterOwnName) }

// CardAddresses is the own address the master holds on the card now (Release keeps it).
func (m *Master) CardAddresses() map[string]string {
	used := map[string]string{}
	if address := m.claim.held(); address != "" {
		used[address] = MasterOwnName
	}
	return used
}

// OwnAddresses are the card addresses the master is set to use (for the lab network page's checks).
func (m *Master) OwnAddresses() map[string]string {
	config := m.Config()
	used := map[string]string{}
	if local := localOf(config); isOwn(config, local) {
		used[local] = MasterOwnName
	}
	return used
}
