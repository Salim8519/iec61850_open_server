package labsetup

import (
	"sort"
	"strings"

	"studio-engine/internal/labnet"
)

// Release takes off the cards the addresses this copy of ScadaBench put there and nothing uses now
// (ADR-0127, ADR-0141). Kept: what another device of this PC holds right now (a running gateway,
// master or SNMP device, Projects.InUse), the running IEDs' addresses and, while the project stays
// open, the addresses this session put on for its IEDs (trial N10). Only addresses this copy's lab
// journal says it added are ever removed: another ScadaBench copy's (installed and development,
// or a second data folder) stay, so two copies no longer break each other's Runs (finding of the
// 2026-11-20 trials). The coordinator calls Release after every Stop, the app at launch, after a
// project change and from Use. It never stops anything: an address it cannot remove stays until
// the next Release or a Windows restart, and the error names each one with its reason (change
// 0221, trial N1).
func (s *Service) Release() error { return s.release(false) }

// EndSession is Release without the session's addresses: at exit and from the lab page's Clean
// up, everything this copy put on a card that nothing uses now comes off (ADR-0141).
func (s *Service) EndSession() error { return s.release(true) }

func (s *Service) release(ended bool) error {
	s.card.Lock()
	defer s.card.Unlock()
	if ended {
		s.session.clear()
	}
	left := s.releaseLab()
	left = append(left, s.releaseReal()...) // approved addresses on Ethernet follow the run too (ADR-0132)
	return releaseLeft(left)
}

func (s *Service) releaseLab() []failure {
	card := s.Windows.Card()
	if card == "" {
		return nil
	}
	keep, names, ok := s.keptLab(card)
	if !ok {
		return nil
	}
	addresses, down := s.labAddressesNow(card)
	ours := s.outstanding("lab_address")
	if down == "" { // only the service's own list proves an address is gone
		s.settleLabGone(card, addresses, ours)
	}
	var off []string
	for _, address := range addresses {
		if ours[realKey(card, address)] && !keep[address] {
			off = append(off, address)
		}
	}
	if len(off) == 0 {
		return nil
	}
	if down == "" {
		failed := s.removeAll(off, card)
		s.session.forgetAll(card, off, failed)
		return failed
	}
	var left []failure
	for _, address := range off { // the service cannot take them off now
		s.journal("failed", "lab_address", address, names[address], card, down)
		left = append(left, failure{address, names[address], labnet.CardName, down})
	}
	return left
}

// keptLab is the Lab-card addresses Release leaves on, and the project's IED names by address.
// ok is false when the project cannot be read (nothing is removed then).
func (s *Service) keptLab(card string) (map[string]bool, map[string]string, bool) {
	keep, names := map[string]bool{}, map[string]string{}
	for address := range s.Projects.InUse() {
		keep[address] = true
	}
	if mode, guid := s.Projects.Network(); mode == "lab" && strings.EqualFold(guid, card) {
		ieds, err := s.Projects.SimulatedIEDs()
		if err != nil {
			return nil, nil, false
		}
		running := s.Runtime.Running()
		for _, ied := range ieds {
			names[ied.LabIP] = ied.Name
			if ied.LabIP != "" && (contains(running, ied.ID) || s.session.has(card, ied.LabIP)) {
				keep[ied.LabIP] = true
			}
		}
	}
	return keep, names, true
}

// settleLabGone journals as settled the addresses this copy added that are no longer on the Lab
// card (Windows restarted, the card was repaired, an older ScadaBench took them off): such an
// address may later be another copy's, and must never be taken for ours.
func (s *Service) settleLabGone(card string, addresses []string, ours map[string]bool) {
	present := map[string]bool{}
	for _, address := range addresses {
		present[address] = true
	}
	gone := []string{}
	for key := range ours {
		if address, found := strings.CutPrefix(key, realKey(card, "")); found && !present[address] {
			gone = append(gone, address)
		}
	}
	sort.Strings(gone)
	for _, address := range gone {
		delete(ours, realKey(card, address))
		s.session.forget(card, address)
		s.journal("skipped", "lab_address", address, "", card,
			"Not on the card (Windows restarted, the card was repaired or another program took it off); nothing to take off.")
	}
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
