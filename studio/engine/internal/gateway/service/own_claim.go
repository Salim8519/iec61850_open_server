package service

import "sync"

// ownClaim is the own address a starting or running gateway or master holds on the ScadaBench Lab
// card (ADR-0141, trial N9): it goes on the card at Start/Connect and comes off at
// Stop/Disconnect and at exit. The claim is visible (held) before the address is put on, so a
// Release that runs meanwhile (an IED or SNMP Stop) keeps it (trial N3).
type ownClaim struct {
	mu      sync.Mutex
	address string
}

// take holds address for owner and puts it on the card; on failure nothing stays held.
func (c *ownClaim) take(lab LabAddresses, address, owner string) error {
	c.mu.Lock()
	previous := c.address
	c.address = address
	c.mu.Unlock()
	if previous != "" && previous != address {
		lab.RemoveOwn(previous, owner)
	}
	if err := lab.EnsureOwn(address, owner); err != nil {
		c.give(lab, owner)
		return err
	}
	return nil
}

// give takes the held address off the card and lets it go.
func (c *ownClaim) give(lab LabAddresses, owner string) {
	c.mu.Lock()
	address := c.address
	c.address = ""
	c.mu.Unlock()
	if address != "" && lab != nil {
		lab.RemoveOwn(address, owner)
	}
}

// held is the address on the card now ("" when none).
func (c *ownClaim) held() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.address
}
