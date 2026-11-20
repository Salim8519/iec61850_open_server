package labsetup

import "studio-engine/internal/labnet"

// Lost answers (change 0221, trial N2): a pipe error or a timeout does not mean nothing changed;
// the service may have added or removed the address before its answer was lost. Before, such a
// request journaled nothing, so a real-card address the service did add was never taken off
// again (Stop removes only what the journal says ScadaBench added, fix plan A6). Now a real-card
// add is journaled "requested" before the request, and after a lost answer the card itself is
// read and every address is journaled as it really is.
//
// Only the service's own list settles a lab-card request (change 0227): the service changes the
// card one request at a time, so its next answer comes after the lost request was done. When it
// does not answer either (still busy with that request), Windows' list may not show the result
// yet: a lab add is then journaled "requested" (it counts as this copy's, ADR-0141) and a removal
// is not counted as done, so the next Release settles the address from the service's list (on
// the card: taken off; not there: "skipped"). Before, such an add was journaled "failed", and
// when the service finished it later the address stayed on the card until Windows restarted.

// notYet ends the journal message of a lost lab request that only the next Release can settle.
const notYet = "; the next release checks the card again"

// lostLabAdds settles lab-card adds whose answer was lost and returns the ones not on the card.
func (s *Service) lostLabAdds(items []wanted, card string, lost error) []failure {
	present, proven := s.labNow(card)
	var failures []failure
	for _, item := range items {
		switch {
		case present[item.address]:
			s.journal("applied", "lab_address", item.address, item.name, card, "")
			continue
		case proven:
			s.journal("failed", "lab_address", item.address, item.name, card, lost.Error())
		default: // the service may still put it on
			s.journal("requested", "lab_address", item.address, item.name, card, lost.Error()+notYet)
		}
		failures = append(failures, failure{item.address, item.name, labnet.CardName, lost.Error()})
	}
	return failures
}

// lostLabRemoves settles lab-card removals whose answer was lost and returns the ones still there.
func (s *Service) lostLabRemoves(addresses []string, card string, lost error) []failure {
	present, proven := s.labNow(card)
	var failures []failure
	for _, address := range addresses {
		switch {
		case !present[address] && proven:
			s.journal("removed", "lab_address", address, "", card, "")
			continue
		case !present[address]: // off as far as Windows shows; the service may still be busy
			s.journal("failed", "lab_address", address, "", card, lost.Error()+notYet)
			continue
		}
		s.journal("failed", "lab_address", address, "", card, lost.Error())
		failures = append(failures, failure{address, "", labnet.CardName, lost.Error()})
	}
	return failures
}

// labNow is the lab card's addresses now and whether the service listed them (proven), else
// Windows' own list.
func (s *Service) labNow(card string) (map[string]bool, bool) {
	if reply, err := s.call(labnet.Request{Op: labnet.OpStatus}); err == nil && reply.OK {
		found := map[string]bool{}
		for _, address := range reply.Addresses {
			found[address] = true
		}
		return found, true
	}
	return s.addressesOn(card), false
}

// lostReal settles a real-card request whose answer was lost from Windows' own list of the card.
// An add stays journaled "requested" when the address is not there yet: the service may still be
// checking it, and Stop then takes it off if it appears (outstandingReal).
func (s *Service) lostReal(op, guid string, addresses []string, names map[string]string, lost error) []failure {
	present := s.addressesOn(guid)
	var failures []failure
	for _, address := range addresses {
		changed := present[address] == (op == labnet.OpAdd)
		switch {
		case changed && op == labnet.OpAdd:
			s.journal("applied", "real_address", address, names[address], guid, "")
		case changed:
			s.journal("removed", "real_address", address, names[address], guid, "")
		default:
			s.journal("failed", "real_address", address, names[address], guid, lost.Error())
			failures = append(failures, failure{address, names[address], "", lost.Error()})
		}
	}
	return failures
}
