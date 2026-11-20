package labsetup

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/labnet"
	"studio-engine/internal/pyjson"
)

func plain(message string) error { return apperr.New(message) }

// ready refuses before the card and service work.
func (s *Service) ready(status Status) error {
	if !status.Ready {
		if status.Problem != "" {
			return plain(status.Problem)
		}
		return plain("Create the lab network card first.")
	}
	return nil
}

// Use is POST /network/lab/use {"addresses": {ied_id: ip}}: every simulated IED keeps its checked
// address and the project switches to the card. The addresses go on the card at Run (ADR-0127).
// Run must be stopped (a network change).
func (s *Service) Use(chosen map[string]string) (Status, error) {
	s.change.Lock()
	defer s.change.Unlock()
	if len(s.Runtime.Running()) > 0 {
		return Status{}, plain("Stop the simulation before switching the network.")
	}
	status, err := s.settled(chosen)
	if err != nil {
		return status, err
	}
	if err := s.ready(status); err != nil {
		return status, err
	}
	if len(status.Rows) == 0 {
		return status, plain("No IED is set to 'Simulate', so there is nothing to put on the lab network.")
	}
	addresses := map[string]string{}
	for _, row := range status.Rows {
		if row.Blocking() {
			return status, plain(row.Name + ": " + row.Hint)
		}
		addresses[row.IEDID] = row.IP
	}
	if err := s.freeOurs(status.Rows); err != nil {
		return status, err
	}
	if err := s.Projects.UseCard(addresses, status.Card, labnet.CardName); err != nil {
		return status, err
	}
	s.Release() // addresses left from before (nothing runs now)
	return s.Status(nil)
}

// SetAddress is PUT /network/lab/address {"ied_id", "ip"}: one IED moves to another address on
// the card at any time. A running IED is stopped and started again on its new address.
func (s *Service) SetAddress(id, address string) (Status, error) {
	s.change.Lock()
	defer s.change.Unlock()
	status, err := s.settled(map[string]string{id: strings.TrimSpace(address)})
	if err != nil {
		return status, err
	}
	if err := s.ready(status); err != nil {
		return status, err
	}
	if !status.Active {
		return status, plain("Put the IEDs on the lab network first (Use the lab network).")
	}
	var row *Row
	for index := range status.Rows {
		if status.Rows[index].IEDID == id {
			row = &status.Rows[index]
		}
	}
	if row == nil {
		return status, apperr.Coded("Unknown IED", "not_found")
	}
	if row.Blocking() {
		return status, plain(row.Hint)
	}
	ieds, err := s.Projects.SimulatedIEDs()
	if err != nil {
		return status, err
	}
	for _, ied := range ieds {
		if ied.ID == id && ied.LabIP == row.IP {
			return s.Status(nil)
		}
	}
	// Stop takes the old address off the card and Start puts the new one on (ADR-0127).
	running := contains(s.Runtime.Running(), id)
	if running {
		if err := s.Runtime.Stop([]string{id}); err != nil {
			return status, err
		}
	}
	if err := s.Projects.SetLabIP(id, row.IP); err != nil {
		return status, s.restart(running, id, err)
	}
	if running {
		if err := s.Runtime.Start([]string{id}); err != nil {
			return status, plain(row.Name + " moved to " + row.IP + " but did not start again: " + err.Error())
		}
	}
	return s.Status(nil)
}

// restart starts an IED that was stopped for a change that then failed, and returns the failure.
func (s *Service) restart(running bool, id string, failure error) error {
	if running {
		s.Runtime.Start([]string{id})
	}
	return failure
}

// EnsureForRun puts the stored addresses of the IEDs about to start on the card (ADR-0127: an
// address is on the card only while its IED runs). Every address that does not go on stops the
// Run with its own reason, and so does a service that is off or does not answer while addresses
// are missing (change 0221); before, Run went on and only the preflight's "not set up" followed.
func (s *Service) EnsureForRun(guid string, starting []string) error {
	s.card.Lock()
	defer s.card.Unlock()
	card := s.Windows.Card()
	if s.needsService(guid) {
		if err := s.turnOnForRun(); err != nil {
			return err
		}
	}
	if card == "" || !strings.EqualFold(card, guid) {
		return s.ensureReal(guid, starting) // a real card with approved addresses (ADR-0132)
	}
	ieds, err := s.Projects.SimulatedIEDs()
	if err != nil {
		return nil
	}
	var needed []wanted
	for _, ied := range ieds {
		if ied.LabIP != "" && contains(starting, ied.ID) {
			needed = append(needed, wanted{ied.LabIP, ied.Name})
		}
	}
	if len(needed) == 0 {
		return nil
	}
	if _, running := s.Windows.State(); !running {
		return serviceDownForRun(s.notOn(card, needed), "is off")
	}
	reply, err := s.call(labnet.Request{Op: labnet.OpStatus})
	if err != nil || !reply.OK {
		return serviceDownForRun(s.notOn(card, needed), "does not answer")
	}
	var missing []wanted
	for _, item := range needed {
		if !contains(reply.Addresses, item.address) {
			missing = append(missing, item)
		}
	}
	return runRefused(s.addAll(missing, card))
}

// notOn is the items whose address Windows does not list on the card.
func (s *Service) notOn(card string, items []wanted) []wanted {
	present, missing := s.addressesOn(card), []wanted{}
	for _, item := range items {
		if !present[item.address] {
			missing = append(missing, item)
		}
	}
	return missing
}

// add asks the service for one address and journals the answer; the error is the reason.
func (s *Service) add(address, name, card string) error {
	if failed := s.addOne(wanted{address, name}, card); failed != nil {
		return plain(failed.reason)
	}
	return nil
}

func (s *Service) addOne(item wanted, card string) *failure {
	reply, err := s.call(labnet.Request{Op: labnet.OpAdd, IP: item.address})
	if err != nil {
		if failures := s.lostLabAdds([]wanted{item}, card, err); len(failures) > 0 {
			return &failures[0]
		}
		return nil
	}
	if !reply.OK || !contains(reply.Addresses, item.address) {
		s.journal("failed", "lab_address", item.address, item.name, card, reply.Message)
		return &failure{item.address, item.name, labnet.CardName, reply.Message}
	}
	s.journal("applied", "lab_address", item.address, item.name, card, "")
	return nil
}

func (s *Service) remove(address, name, card string) { s.removeOne(address, name, card) }

func (s *Service) removeOne(address, name, card string) *failure {
	reply, err := s.call(labnet.Request{Op: labnet.OpRemove, IP: address})
	if err != nil {
		if failures := s.lostLabRemoves([]string{address}, card, err); len(failures) > 0 {
			return &failures[0]
		}
		return nil
	}
	event := "removed"
	if !reply.OK {
		event = "failed"
	}
	s.journal(event, "lab_address", address, name, card, reply.Message)
	if !reply.OK {
		return &failure{address, name, labnet.CardName, reply.Message}
	}
	return nil
}

// journal appends one record to network/lab_journal.jsonl (append-only, never rewritten).
func (s *Service) journal(event, kind, address, name, card, message string) {
	if s.Journal == nil {
		return
	}
	item := pyjson.NewObject().Set("kind", kind).Set("card", card)
	if address != "" {
		item.Set("ip", address)
	}
	if name != "" {
		item.Set("for", name)
	}
	s.Journal.Append(event, item, "", message)
}

func contains(list []string, wanted string) bool {
	for _, item := range list {
		if item == wanted {
			return true
		}
	}
	return false
}
