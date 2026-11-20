package labsetup

import (
	"sort"
	"strings"

	"studio-engine/internal/labnet"
)

// Release takes off the card every address that no running IED and no other device of this PC
// (the gateway or the master, ADR-0125) uses (ADR-0127): an IED's address is on the card only
// while it runs. Run puts it on (EnsureForRun); the coordinator calls Release after every Stop,
// and the app at launch and exit. It never stops anything: an address it cannot remove stays
// until the next Release or a Windows restart, and the error names each one with its reason
// (change 0221, trial N1; before, a stopped service made Release silently do nothing).
func (s *Service) Release() error {
	s.card.Lock()
	defer s.card.Unlock()
	left := s.releaseLab()
	left = append(left, s.releaseReal()...) // approved addresses on Ethernet follow the run too (ADR-0132)
	return releaseLeft(left)
}

func (s *Service) releaseLab() []failure {
	card := s.Windows.Card()
	if card == "" {
		return nil
	}
	addresses, down := s.labAddressesNow(card)
	if len(addresses) == 0 {
		return nil
	}
	keep := map[string]bool{}
	for address := range s.Projects.Reserved() {
		keep[address] = true
	}
	names := map[string]string{}
	if mode, guid := s.Projects.Network(); mode == "lab" && strings.EqualFold(guid, card) {
		ieds, err := s.Projects.SimulatedIEDs()
		if err != nil {
			return nil
		}
		running := s.Runtime.Running()
		for _, ied := range ieds {
			names[ied.LabIP] = ied.Name
			if ied.LabIP != "" && contains(running, ied.ID) {
				keep[ied.LabIP] = true
			}
		}
	}
	var off []string
	for _, address := range addresses {
		// Only addresses the service could have added: Windows gives the card itself a 169.254.x
		// address, which the service refuses (ADR-0128).
		if _, err := labnet.ParseAddress(address); err == nil && !keep[address] {
			off = append(off, address)
		}
	}
	if down == "" {
		return s.removeAll(off, card)
	}
	var left []failure
	for _, address := range off { // the service cannot take them off now
		s.journal("failed", "lab_address", address, names[address], card, down)
		left = append(left, failure{address, names[address], labnet.CardName, down})
	}
	return left
}

// labAddressesNow is the lab card's addresses from the service, or from Windows' own list with
// the reason the service cannot change them now ("" while it answers).
func (s *Service) labAddressesNow(card string) ([]string, string) {
	down := "the lab network service is off"
	if _, running := s.Windows.State(); running {
		reply, err := s.call(labnet.Request{Op: labnet.OpStatus})
		if err == nil && reply.OK {
			return reply.Addresses, ""
		}
		down = "the lab network service does not answer"
	}
	addresses := []string{}
	for address := range s.addressesOn(card) {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return addresses, down
}
