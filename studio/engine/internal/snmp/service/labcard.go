package service

import "studio-engine/internal/snmp/model"

// SNMP devices on the ScadaBench Lab card (change 0213): like the station gateway's own address
// (ADR-0125), a device's lab address must be on the card before its agent binds, and stays there
// only while the device runs. On a real card Network setup's plan already put it there.

// LabCard is the lab network as the SNMP devices use it; a nil func does nothing.
type LabCard struct {
	// Ensure puts address on the card for owner when the project uses the Lab card.
	Ensure func(address, owner string) error
	// Release takes addresses nobody uses any more off the card.
	Release func()
}

// SetLabCard joins the devices to the Lab card.
func (s *Service) SetLabCard(card LabCard) {
	s.mu.Lock()
	s.labCard = card
	s.mu.Unlock()
}

// CardOwner is the name a device's address carries in the lab journal and the clash checks.
func CardOwner(name string) string { return "SNMP · " + name }

// ensureOnCard puts a placed device's lab address on the card before the agent binds it.
func (s *Service) ensureOnCard(placed model.Device) error {
	s.mu.Lock()
	ensure := s.labCard.Ensure
	s.mu.Unlock()
	if ensure == nil {
		return nil
	}
	return ensure(placed.IP, CardOwner(placed.Name))
}

// CardAddresses are the lab addresses running devices answer at now (address -> owner), so the
// card keeps them while the IEDs stop and start.
func (s *Service) CardAddresses() map[string]string {
	used := map[string]string{}
	if !s.onLab() {
		return used
	}
	names := map[string]string{}
	for _, device := range s.Config().Devices {
		names[device.ID] = device.Name
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, state := range s.devices {
		if address := s.bound[id]; state.State == "running" && address != "" {
			used[address] = CardOwner(names[id])
		}
	}
	return used
}

// releaseCard takes the addresses of stopped devices off the card.
func (s *Service) releaseCard() {
	s.mu.Lock()
	release := s.labCard.Release
	s.mu.Unlock()
	if release != nil {
		release()
	}
}
