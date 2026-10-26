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

// Use is POST /network/lab/use {"addresses": {ied_id: ip}}: every simulated IED gets its address
// on the card and the project switches to the card. Run must be stopped (a network change).
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
	wanted, addresses := map[string]bool{}, map[string]string{}
	for _, row := range status.Rows {
		if row.Blocking() {
			return status, plain(row.Name + ": " + row.Hint)
		}
		wanted[row.IP], addresses[row.IEDID] = true, row.IP
	}
	for _, row := range status.Rows {
		if err := s.add(row.IP, row.Name, status.Card); err != nil {
			return status, err
		}
	}
	for _, address := range status.CardIPs {
		if !wanted[address] {
			s.remove(address, "", status.Card) // an address no IED uses any more; failure is harmless
		}
	}
	if err := s.Projects.UseCard(addresses, status.Card, labnet.CardName); err != nil {
		return status, err
	}
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
	old, users := "", map[string]int{}
	for _, ied := range ieds {
		if ied.ID == id {
			old = ied.LabIP
		} else if ied.LabIP != "" {
			users[ied.LabIP]++
		}
	}
	if old == row.IP {
		return s.Status(nil)
	}
	running := contains(s.Runtime.Running(), id)
	if running {
		if err := s.Runtime.Stop([]string{id}); err != nil {
			return status, err
		}
	}
	if err := s.add(row.IP, row.Name, status.Card); err != nil {
		return status, s.restart(running, id, err)
	}
	if err := s.Projects.SetLabIP(id, row.IP); err != nil {
		return status, s.restart(running, id, err)
	}
	if old != "" && users[old] == 0 {
		s.remove(old, row.Name, status.Card)
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

// EnsureForRun puts every simulated IED's stored address back on the card before Run (they are
// gone after Windows restarts). It does nothing unless the project uses the card and the service
// answers; the usual Run check reports anything still missing.
func (s *Service) EnsureForRun(guid string) {
	card := s.Windows.Card()
	if card == "" || !strings.EqualFold(card, guid) {
		return
	}
	if _, running := s.Windows.State(); !running {
		return
	}
	ieds, err := s.Projects.SimulatedIEDs()
	if err != nil {
		return
	}
	reply, err := s.Windows.Call(labnet.Request{Op: labnet.OpStatus})
	if err != nil || !reply.OK {
		return
	}
	for _, ied := range ieds {
		if ied.LabIP != "" && !contains(reply.Addresses, ied.LabIP) {
			s.add(ied.LabIP, ied.Name, card)
		}
	}
}

// add asks the service for one address and journals the answer.
func (s *Service) add(address, name, card string) error {
	reply, err := s.Windows.Call(labnet.Request{Op: labnet.OpAdd, IP: address})
	if err != nil {
		return plain(err.Error())
	}
	present := contains(reply.Addresses, address)
	if !reply.OK || !present {
		s.journal("failed", "lab_address", address, name, card, reply.Message)
		return plain(reply.Message)
	}
	s.journal("applied", "lab_address", address, name, card, "")
	return nil
}

func (s *Service) remove(address, name, card string) {
	reply, err := s.Windows.Call(labnet.Request{Op: labnet.OpRemove, IP: address})
	if err != nil {
		return
	}
	event := "removed"
	if !reply.OK {
		event = "failed"
	}
	s.journal(event, "lab_address", address, name, card, reply.Message)
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
