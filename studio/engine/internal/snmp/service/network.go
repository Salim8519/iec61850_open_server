package service

import (
	"strconv"

	"studio-engine/internal/apperr"
	"studio-engine/internal/snmp/model"
)

// SNMP devices follow the IEDs (ADR-0092 amendment): while the project is on the lab network
// each device answers on the address Network setup gave it (and on its 127.x address, so this
// PC still reaches it and receives its traps); otherwise on its 127.x address only.

// NoLabAddress is the device's message when the project is on the lab network but Network
// setup has not given this device an address yet.
func NoLabAddress(name string) string {
	return name + " has no address on the lab network yet. Open Settings › Network and put the " +
		"devices on the network again: ScadaBench finds a free address for each SNMP device."
}

// SetNetwork tells the service whether the project is on the lab network (true) or uses
// this PC only; nil means this PC only.
func (s *Service) SetNetwork(lab func() bool) {
	s.mu.Lock()
	s.network = lab
	s.mu.Unlock()
}

func (s *Service) onLab() bool {
	s.mu.Lock()
	lab := s.network
	s.mu.Unlock()
	return lab != nil && lab()
}

// place is the device as the worker runs it: on its lab address while the project is on the
// lab network, with its own 127.x address as the twin.
func place(device model.Device, lab bool) (model.Device, error) {
	if !lab {
		return device, nil
	}
	if device.LabIP == "" {
		return device, apperr.ValueError(NoLabAddress(device.Name))
	}
	device.LocalIP, device.IP = device.IP, device.LabIP
	return device, nil
}

// answersAt is the address a device answers at (or will once started), for the device list.
func answersAt(device model.Device, lab bool) string {
	if lab && device.LabIP != "" {
		return device.LabIP + ":" + strconv.Itoa(device.Port)
	}
	return device.IP + ":" + strconv.Itoa(device.Port)
}

// LabDevice is one device as Network setup plans it.
type LabDevice struct {
	ID, Name, LabIP string
	Port            int
}

// LabDevices lists every SNMP device for Network setup.
func (s *Service) LabDevices() []LabDevice {
	devices := []LabDevice{}
	for _, device := range s.Config().Devices {
		devices = append(devices, LabDevice{ID: device.ID, Name: device.Name, LabIP: device.LabIP, Port: device.Port})
	}
	return devices
}

// SetLabIPs saves the addresses Network setup gave the devices (device id -> address).
// Running devices move to them when the project is on the lab network.
func (s *Service) SetLabIPs(addresses map[string]string) error {
	config := s.Config()
	for index, device := range config.Devices {
		if address, found := addresses[device.ID]; found {
			config.Devices[index].LabIP = address
		}
	}
	if err := s.save(config); err != nil {
		return err
	}
	s.NetworkChanged()
	return nil
}

// NetworkChanged restarts running devices whose address changed (the project moved between
// this PC and the lab network, or Network setup gave new addresses).
func (s *Service) NetworkChanged() {
	if !s.Running() {
		s.changed(ChangeStatus)
		return
	}
	lab := s.onLab()
	s.mu.Lock()
	bound := s.bound
	s.mu.Unlock()
	moved := false
	for _, device := range s.Config().Devices {
		placed, err := place(device, lab)
		if err != nil || bound[device.ID] != placed.IP {
			moved = true
		}
	}
	if !moved {
		return
	}
	go func() {
		if _, err := s.Start(); err != nil {
			s.logf("SNMP devices did not move to their new addresses: %v", err)
		}
	}()
}
