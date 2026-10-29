package labsetup

import (
	"strings"

	"studio-engine/internal/labnet"
)

// Own addresses (ADR-0125): a device of this PC that is not an IED, such as the station gateway,
// may answer at its own address on the ScadaBench Lab card, exactly as an IED does. The same
// rules apply: a unicast IPv4 address no IED, no other card and no real device on a connected
// card uses. Addresses on the card are gone after Windows restarts, so the owner ensures its
// address again before it starts.

// Ready says the card and the lab network service can take an address now (no network reads).
func (s *Service) Ready() bool {
	installed, running := s.Windows.State()
	return installed && running && s.Windows.Card() != ""
}

// AddOwn checks address for owner (a plain name such as "Station gateway") and puts it on the
// card. It returns the address as ScadaBench writes it.
func (s *Service) AddOwn(address, owner string) (string, error) {
	s.change.Lock()
	defer s.change.Unlock()
	parsed, err := labnet.ParseAddress(strings.TrimSpace(address))
	if err != nil {
		return "", plain("Not an address ScadaBench can use: " + err.Error() + ".")
	}
	status, err := s.Status(nil)
	if err != nil {
		return "", err
	}
	if err := s.ready(status); err != nil {
		return "", err
	}
	if err := s.ownClash(parsed, owner, status); err != nil {
		return "", err
	}
	return parsed, s.add(parsed, owner, status.Card)
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

// EnsureOwn puts an address that was checked before back on the card (after a Windows restart).
func (s *Service) EnsureOwn(address, owner string) error {
	s.change.Lock()
	defer s.change.Unlock()
	card := s.Windows.Card()
	if card == "" {
		return plain("The ScadaBench Lab card is missing, so " + address + " cannot be used. " +
			"Create it on the lab network page, or choose another network.")
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

// RemoveOwn takes an address its owner no longer uses off the card (failure is harmless).
func (s *Service) RemoveOwn(address, owner string) {
	s.change.Lock()
	defer s.change.Unlock()
	if card := s.Windows.Card(); card != "" {
		s.remove(address, owner, card)
	}
}
