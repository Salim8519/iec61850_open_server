// Package netaddr sets one simulated IED's or SNMP device's network address from where the user
// sees it, the Devices page or the SNMP device editor (change 0215), without a trip to Network
// setup. The project's network decides how: on the ScadaBench Lab card the lab page's own checks
// apply and nothing asks Windows; on a real card the address plan is reviewed again with this one
// address changed and applied, so the network service may add it (one Windows prompt, ADR-0132,
// ADR-0136); on "This PC only" there is no network address to set.
package netaddr

import (
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/netcfg"
	"studio-engine/internal/pyjson"
)

// Device is one simulated IED or SNMP device with the address it has on the network now.
type Device struct {
	ID, Name, LabIP string
}

// Deps are the services the address change goes through.
type Deps struct {
	Network func() (mode, guid string) // the project's network mode and card
	LabCard func() string              // the ScadaBench Lab card's GUID ("" when none)
	Running func() []string            // the IEDs running now
	IEDs    func() ([]Device, error)   // simulated IEDs
	SNMP    func() []Device            // SNMP devices
	// The lab card: an IED moves at once (restarted when running); a device's address is checked.
	LabSetIED   func(id, ip string) error
	LabCheckOwn func(ip, owner string) (string, error)
	SetSNMPLab  func(id, ip string) error
	// A real card: the plan with typed addresses, then Apply (one Windows prompt).
	Preview  func(guid any, strategy string, manual any) (*pyjson.Object, error)
	Apply    func(fingerprint, confirm any) (*pyjson.Object, error)
	Adapters func() ([]netcfg.Adapter, error)
}

// Service is the address change.
type Service struct{ Deps }

// New returns the service over deps.
func New(deps Deps) *Service { return &Service{deps} }

// SetIED gives a simulated IED a new address on the project's network.
func (s *Service) SetIED(id, ip string) (*pyjson.Object, error) {
	ip = strings.TrimSpace(ip)
	ieds, err := s.IEDs()
	if err != nil {
		return nil, err
	}
	device, found := find(ieds, id)
	if !found {
		return nil, apperr.Coded("That IED is not simulated in this project.", "not_found")
	}
	mode, guid := s.Network()
	switch {
	case mode != "lab" || guid == "":
		return nil, apperr.New(thisPCOnly(device.Name))
	case strings.EqualFold(guid, s.LabCard()):
		if err := s.LabSetIED(id, ip); err != nil {
			return nil, err
		}
		return done(device.Name + " now answers at " + ip + " on ScadaBench Lab."), nil
	}
	if len(s.Running()) > 0 {
		return nil, apperr.New("Stop the simulation first: a new address on a network card is approved " +
			"once by Windows, then used from the next Start.")
	}
	return s.real(guid, id, ip, device.Name, "from the next Start")
}

// SetSNMP gives an SNMP device a new address on the project's network.
func (s *Service) SetSNMP(id, ip string) (*pyjson.Object, error) {
	ip = strings.TrimSpace(ip)
	device, found := find(s.SNMP(), id)
	if !found {
		return nil, apperr.Coded("That SNMP device is not in this project.", "not_found")
	}
	mode, guid := s.Network()
	switch {
	case mode != "lab" || guid == "":
		return nil, apperr.New(thisPCOnly(device.Name))
	case strings.EqualFold(guid, s.LabCard()):
		parsed, err := s.LabCheckOwn(ip, "SNMP · "+device.Name)
		if err != nil {
			return nil, err
		}
		if other, taken := takenBySNMP(s.SNMP(), id, parsed); taken {
			return nil, apperr.New("SNMP device " + other + " uses " + parsed + ". Type another address.")
		}
		if err := s.SetSNMPLab(id, parsed); err != nil {
			return nil, err
		}
		return done(device.Name + " now answers at " + parsed + " on ScadaBench Lab."), nil
	}
	if len(s.Running()) > 0 {
		return nil, apperr.New("Stop the simulation first: Windows approves the new address once while " +
			"nothing runs.")
	}
	return s.real(guid, netcfg.SNMPRowPrefix+id, ip, device.Name, "when the SNMP devices start")
}

func thisPCOnly(name string) string {
	return name + " runs on this PC only, at an address ScadaBench picks. To give it an address of " +
		"your own, put the project on a network first (Settings › Network)."
}

func done(message string) *pyjson.Object { return pyjson.NewObject().Set("message", message) }

func find(devices []Device, id string) (Device, bool) {
	for _, device := range devices {
		if device.ID == id {
			return device, true
		}
	}
	return Device{}, false
}

func takenBySNMP(devices []Device, id, ip string) (string, bool) {
	for _, device := range devices {
		if device.ID != id && device.LabIP == ip {
			return device.Name, true
		}
	}
	return "", false
}
