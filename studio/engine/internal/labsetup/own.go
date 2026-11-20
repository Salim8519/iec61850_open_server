package labsetup

import (
	"strings"

	"studio-engine/internal/labnet"
)

// Own addresses (ADR-0125): a device of this PC that is not an IED, such as the station gateway,
// may answer at its own address on the ScadaBench Lab card, exactly as an IED does. The same
// rules apply: a unicast IPv4 address no IED, no other card and no real device on a connected
// card uses. The address is only checked and stored when it is typed; it goes on the card when
// its owner starts (EnsureOwn) and comes off when it stops (RemoveOwn; ADR-0141, trial N9).
// While it is on the card the owner reports it (Projects.InUse), so a Release never takes it off.

// Ready says the card and the lab network service can take an address now (no network reads).
func (s *Service) Ready() bool {
	installed, running := s.Windows.State()
	return installed && running && s.Windows.Card() != ""
}

// CheckOwn checks address for owner (a plain name such as "Station gateway") without putting it
// on the card, and returns the address as ScadaBench writes it. The owner stores it first and
// puts it on at its start (ADR-0141; an SNMP device since change 0215).
func (s *Service) CheckOwn(address, owner string) (string, error) {
	s.change.Lock()
	defer s.change.Unlock()
	parsed, _, err := s.checkOwn(address, owner)
	return parsed, err
}

func (s *Service) checkOwn(address, owner string) (string, Status, error) {
	parsed, err := labnet.ParseAddress(strings.TrimSpace(address))
	if err != nil {
		return "", Status{}, plain("Not an address ScadaBench can use: " + err.Error() + ".")
	}
	status, err := s.Status(nil)
	if err != nil {
		return "", status, err
	}
	if err := s.ready(status); err != nil {
		return "", status, err
	}
	return parsed, status, s.ownClash(parsed, owner, status)
}

// ownClash refuses an address an IED, another device of this PC, another card or a real device
// on a connected card already uses.
func (s *Service) ownClash(address, owner string, status Status) error {
	for _, row := range status.Rows {
		if row.IP == address {
			return plain("IED " + row.Name + " uses " + address + ". Type another address.")
		}
	}
	for used, name := range s.Projects.Reserved() {
		if used == address && name != owner {
			return plain(name + " uses " + address + ". Type another address.")
		}
	}
	adapters, err := s.Windows.Adapters()
	if err != nil {
		return err
	}
	for _, adapter := range adapters {
		for _, item := range adapter.Addresses {
			if item.IP == address && !strings.EqualFold(adapter.GUID, status.Card) {
				return plain(address + " is the address of “" + adapter.Name + "” on this PC. Choose it in the list instead.")
			}
		}
	}
	for _, real := range status.Reals {
		if within(address, real.Networks) == "" || !real.Connected || real.Source == "" {
			continue
		}
		pair := [][2]string{{address, real.Source}}
		s.Asker.Wait(pair)
		if answered, _ := s.Asker.Answer(address, real.Source); answered {
			return plain("A real device on " + real.Name + " uses " + address + ". Type another address.")
		}
	}
	return nil
}

// EnsureOwn puts an address that was checked before on the card as its owner starts. The owner
// reports it in Projects.InUse before calling, so no Release in between takes it off (N3).
func (s *Service) EnsureOwn(address, owner string) error {
	s.card.Lock()
	defer s.card.Unlock()
	card := s.Windows.Card()
	if card == "" {
		return plain("The ScadaBench Lab card is missing, so " + address + " cannot be used. " +
			"Create it on the lab network page, or choose another network.")
	}
	if err := s.turnOnForRun(); err != nil {
		return err
	}
	reply, err := s.call(labnet.Request{Op: labnet.OpStatus})
	if err != nil {
		return plain("The lab network service does not answer, so " + address + " cannot be used. " +
			"Press Repair on the lab network page, or choose another network.")
	}
	if reply.OK && contains(reply.Addresses, address) {
		return nil
	}
	return s.add(address, owner, card)
}

// reserve blocks IED rows whose address another device of this PC uses.
func reserve(rows []Row, reserved map[string]string) {
	for index := range rows {
		if name, found := reserved[rows[index].IP]; found {
			rows[index].Status = "duplicate"
			rows[index].Hint = name + " uses " + rows[index].IP + ". Each IED needs its own address."
		}
	}
}

// RemoveOwn takes an address its owner stopped using off the card when this copy put it there
// (ADR-0141; failure is harmless, the next Release tries again).
func (s *Service) RemoveOwn(address, owner string) {
	s.card.Lock()
	defer s.card.Unlock()
	if card := s.Windows.Card(); card != "" && s.outstanding("lab_address")[realKey(card, address)] {
		s.remove(address, owner, card)
	}
}
