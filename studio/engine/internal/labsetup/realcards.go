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
// request; Windows checks the network for each before the workers start.
func (s *Service) ensureReal(guid string, starting []string) {
	if s.Windows.Approved == nil {
		return
	}
	approved := s.Windows.Approved()[strings.ToUpper(guid)]
	if len(approved) == 0 || !s.RealCardsReady(guid) {
		return
	}
	ieds, err := s.Projects.SimulatedIEDs()
	if err != nil {
		return
	}
	names, addresses := map[string]string{}, []string{}
	for _, ied := range ieds {
		if ied.LabIP != "" && approved[ied.LabIP] > 0 && contains(starting, ied.ID) {
			names[ied.LabIP], addresses = ied.Name, append(addresses, ied.LabIP)
		}
	}
	if len(addresses) > 0 {
		s.changeReal(labnet.OpAdd, guid, addresses, names)
	}
}

// releaseReal takes approved addresses off every real card unless a running IED of the project's
// card uses them: after Stop, at launch (a crash left them) and at exit.
func (s *Service) releaseReal() {
	if s.Windows.Approved == nil {
		return
	}
	approved := s.Windows.Approved()
	if len(approved) == 0 {
		return
	}
	adapters, err := s.Windows.Adapters()
	if err != nil {
		return
	}
	keep := map[string]bool{}
	if mode, guid := s.Projects.Network(); mode == "lab" {
		ieds, _ := s.Projects.SimulatedIEDs()
		running := s.Runtime.Running()
		for _, ied := range ieds {
			if ied.LabIP != "" && contains(running, ied.ID) {
				keep[strings.ToUpper(guid)+" "+ied.LabIP] = true
			}
		}
	}
	for _, adapter := range adapters {
		guid := strings.ToUpper(adapter.GUID)
		var off []string
		for _, address := range adapter.Addresses {
			if approved[guid][address.IP] > 0 && !keep[guid+" "+address.IP] {
				off = append(off, address.IP)
			}
		}
		if len(off) > 0 && s.RealCardsReady(adapter.GUID) {
			s.changeReal(labnet.OpRemove, adapter.GUID, off, nil)
		}
	}
}

// changeReal sends one add or remove for a real card and journals each address.
func (s *Service) changeReal(op, guid string, addresses []string, names map[string]string) {
	reply, err := s.call(labnet.Request{Op: op, IPs: addresses, Card: guid})
	if err != nil {
		return
	}
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
	}
}
