package labsetup

import (
	"strings"

	"studio-engine/internal/labnet"
)

// Release takes off the card every address that no running IED and no other device of this PC
// (the gateway or the master, ADR-0125) uses (ADR-0127): an IED's address is on the card only
// while it runs. Run puts it on (EnsureForRun); the coordinator calls Release after every Stop,
// and the app at launch and exit. It never fails: an address it cannot remove stays until the
// next Release or a Windows restart.
func (s *Service) Release() {
	s.card.Lock()
	defer s.card.Unlock()
	card := s.Windows.Card()
	if card == "" {
		return
	}
	if _, running := s.Windows.State(); !running {
		return
	}
	reply, err := s.call(labnet.Request{Op: labnet.OpStatus})
	if err != nil || !reply.OK || len(reply.Addresses) == 0 {
		return
	}
	keep := map[string]bool{}
	for address := range s.Projects.Reserved() {
		keep[address] = true
	}
	if mode, guid := s.Projects.Network(); mode == "lab" && strings.EqualFold(guid, card) {
		ieds, err := s.Projects.SimulatedIEDs()
		if err != nil {
			return
		}
		running := s.Runtime.Running()
		for _, ied := range ieds {
			if ied.LabIP != "" && contains(running, ied.ID) {
				keep[ied.LabIP] = true
			}
		}
	}
	var off []string
	for _, address := range reply.Addresses {
		// Only addresses the service could have added: Windows gives the card itself a 169.254.x
		// address, which the service refuses (ADR-0128).
		if _, err := labnet.ParseAddress(address); err == nil && !keep[address] {
			off = append(off, address)
		}
	}
	s.removeAll(off, card)
}
