package labsetup

import (
	"strings"

	"studio-engine/internal/labnet"
)

// Real cards (ADR-0132): when the project's IEDs are on Ethernet (a real card) and their addresses
// were approved at Apply, the ScadaBench network service puts them on at Start and takes them off
// at Stop, without a prompt, exactly as on the lab card (ADR-0127).

// RealCardsReady says whether the service can change approved addresses on this real card now
// (running, protocol 3). The lab card is not a real card.
func (s *Service) RealCardsReady(guid string) bool {
	if guid == "" || s.Windows.Approved == nil {
		return false
	}
	if card := s.Windows.Card(); card != "" && strings.EqualFold(card, guid) {
		return false
	}
	if _, running := s.Windows.State(); !running {
		return false
	}
	if !labnet.RealCards(s.serviceVersion()) {
		s.call(labnet.Request{Op: labnet.OpStatus}) // learn the version once
	}
	return labnet.RealCards(s.serviceVersion())
}

// ensureReal puts the approved addresses of the IEDs about to start on the real card, in one
// request; Windows checks the network for each before the workers start. Refused addresses stop
// the Run with the service's reason for every one of them (another device answers there, the
// cable is out; change 0221), not the preflight's "not set up" that would follow.
func (s *Service) ensureReal(guid string, starting []string) error {
	if s.Windows.Approved == nil {
		return nil
	}
	approved := s.Windows.Approved()[strings.ToUpper(guid)]
	if len(approved) == 0 || !s.RealCardsReady(guid) {
		return nil
	}
	ieds, err := s.Projects.SimulatedIEDs()
	if err != nil {
		return nil
	}
	names, addresses, on := map[string]string{}, []string{}, s.addressesOn(guid)
	for _, ied := range ieds { // an address already on the card is not ScadaBench's to add (A6)
		if ied.LabIP != "" && approved[ied.LabIP] > 0 && contains(starting, ied.ID) && !on[ied.LabIP] {
			names[ied.LabIP], addresses = ied.Name, append(addresses, ied.LabIP)
		}
	}
	if len(addresses) == 0 {
		return nil
	}
	return runRefused(s.changeReal(labnet.OpAdd, guid, addresses, names)) // in project order
}

// EnsureRealOwn puts one approved address of a device that is not an IED (an SNMP device) on the
// real card before it starts (ADR-0136). Nothing happens on a card the service cannot change
// now: the plan never adds such addresses any other way.
func (s *Service) EnsureRealOwn(guid, address, owner string) error {
	s.card.Lock()
	defer s.card.Unlock()
	if s.onCard(guid, address) {
		return nil // an older plan's address, still there until Clean up or a restart
	}
	if s.Windows.Approved == nil || s.Windows.Approved()[strings.ToUpper(guid)][address] == 0 {
		return plain(owner + ": " + address + " is not approved for this card. Open Network setup and " +
			"apply the address plan.")
	}
	if err := s.turnOnForRun(); err != nil {
		return err
	}
	if !s.RealCardsReady(guid) {
		return plain("The ScadaBench network helper is not running, so " + address + " cannot be used. " +
			"Open Network › This PC lab network and press Repair.")
	}
	return runRefused(s.changeReal(labnet.OpAdd, guid, []string{address}, map[string]string{address: owner}))
}

// onCard says whether the real card has address now.
func (s *Service) onCard(guid, address string) bool {
	return s.addressesOn(guid)[address]
}

// releaseReal takes the approved addresses ScadaBench added (journaled, A6) off every real card
// unless a running IED of the project's card uses them: after Stop, at launch (a crash left them)
// and at exit. The addresses that stay come back with their reason (change 0221).
func (s *Service) releaseReal() []failure {
	if s.Windows.Approved == nil {
		return nil
	}
	approved := s.Windows.Approved()
	if len(approved) == 0 {
		return nil
	}
	adapters, err := s.Windows.Adapters()
	if err != nil {
		return nil
	}
	keep := map[string]bool{}
	if mode, guid := s.Projects.Network(); mode == "lab" {
		for address := range s.Projects.Reserved() { // running SNMP devices (ADR-0136)
			keep[strings.ToUpper(guid)+" "+address] = true
		}
		ieds, _ := s.Projects.SimulatedIEDs()
		running := s.Runtime.Running()
		for _, ied := range ieds {
			if ied.LabIP != "" && contains(running, ied.ID) {
				keep[strings.ToUpper(guid)+" "+ied.LabIP] = true
			}
		}
	}
	outstanding, left := s.outstandingReal(), []failure{}
	for _, adapter := range adapters {
		guid := strings.ToUpper(adapter.GUID)
		s.settleGone(adapter, outstanding)
		var off []string
		for _, address := range adapter.Addresses {
			// The service's own addresses are never source addresses; the card's own address is.
			ours := outstanding[realKey(guid, address.IP)] && address.SkipAsSource
			if approved[guid][address.IP] > 0 && ours && !keep[guid+" "+address.IP] {
				off = append(off, address.IP)
			}
		}
		if len(off) == 0 {
			continue
		}
		failed := []failure{}
		if s.RealCardsReady(adapter.GUID) {
			failed = s.changeReal(labnet.OpRemove, adapter.GUID, off, nil)
		} else {
			for _, address := range off {
				failed = append(failed, failure{address: address, reason: "the ScadaBench network service is off"})
			}
		}
		for _, item := range failed {
			item.card = adapter.Name
			left = append(left, item)
		}
	}
	return left
}

// changeReal sends one add or remove for a real card, journals each address and returns the
// failed ones with their reason, in order. An add is journaled "requested" first, and a lost
// answer is settled from Windows' own list of the card (change 0221, trial N2).
func (s *Service) changeReal(op, guid string, addresses []string, names map[string]string) []failure {
	if op == labnet.OpAdd {
		for _, address := range addresses {
			s.journal("requested", "real_address", address, names[address], guid, "")
		}
	}
	reply, err := s.call(labnet.Request{Op: op, IPs: addresses, Card: guid})
	if err != nil {
		return s.lostReal(op, guid, addresses, names, err)
	}
	var failed []failure
	done := map[string]string{labnet.OpAdd: "applied", labnet.OpRemove: "removed"}[op]
	for _, address := range addresses {
		event, reason := done, reply.Failed[address]
		if reason != "" || (!reply.OK && reply.Failed == nil) {
			event = "failed"
		}
		if event == "failed" && reason == "" {
			reason = reply.Message
		}
		s.journal(event, "real_address", address, names[address], guid, reason)
		if event == "failed" {
			failed = append(failed, failure{address: address, name: names[address], reason: reason})
		}
	}
	return failed
}
