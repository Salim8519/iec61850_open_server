package service

import (
	"fmt"
	"strings"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/profiles"
)

// startTimeout covers binding every device and localizing their v3 keys.
const startTimeout = 60 * time.Second

// Running reports whether simulated devices run.
func (s *Service) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, state := range s.devices {
		if state.State == "running" {
			return true
		}
	}
	return false
}

// AddDevice adds a device of a profile with working defaults.
func (s *Service) AddDevice(profile string) (model.Device, error) {
	if profiles.Points(profile, "x") == nil {
		return model.Device{}, apperr.ValueError("Choose Network switch, GPS time server or Empty device")
	}
	config := s.Config()
	device := NewDevice(profile, config.Devices)
	config.Devices = append(config.Devices, device)
	return device, s.saveAndRestart(config)
}

// UpdateDevice replaces a device's settings; its points stay unless points is true.
func (s *Service) UpdateDevice(device model.Device, points bool) (model.Device, error) {
	config := s.Config()
	index, err := config.findDevice(device.ID)
	if err != nil {
		return model.Device{}, err
	}
	device.Name = strings.TrimSpace(device.Name)
	if !points {
		device.Points = config.Devices[index].Points
	}
	if device.Profile == "" {
		device.Profile = config.Devices[index].Profile
	}
	if device.Users == nil {
		device.Users = []model.User{}
	}
	if device.Traps == nil {
		device.Traps = []model.TrapTarget{}
	}
	if err := CheckDevice(device, config.Devices); err != nil {
		return model.Device{}, apperr.ValueError(err.Error())
	}
	config.Devices[index] = device
	return device, s.saveAndRestart(config)
}

// DeleteDevice removes a device (it stops answering at once).
func (s *Service) DeleteDevice(id string) error {
	config := s.Config()
	index, err := config.findDevice(id)
	if err != nil {
		return apperr.ValueError(err.Error())
	}
	config.Devices = append(config.Devices[:index], config.Devices[index+1:]...)
	return s.saveAndRestart(config)
}

// saveAndRestart saves the config; running devices restart with it.
func (s *Service) saveAndRestart(config Config) error {
	if err := s.save(config); err != nil {
		return err
	}
	if s.Running() {
		_, err := s.Start()
		return err
	}
	return nil
}

// Start starts every device of the project. Devices that cannot start are reported one by
// one; the others run.
func (s *Service) Start() (*Status, error) {
	config := s.Config()
	valid := []model.Device{}
	states := map[string]deviceState{}
	for _, device := range config.Devices {
		if err := CheckDevice(device, config.Devices); err != nil {
			states[device.ID] = deviceState{State: "failed", Error: err.Error()}
			continue
		}
		valid = append(valid, device)
	}
	if len(config.Devices) == 0 {
		return nil, apperr.ValueError("Add a device first: press Add device")
	}
	answer, err := s.call("agents_start", map[string]any{"devices": valid}, startTimeout)
	if err != nil {
		return nil, err
	}
	var started struct {
		Started []struct {
			ID    string `json:"id"`
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		} `json:"started"`
	}
	if err := FromPy(answer, &started); err != nil {
		return nil, err
	}
	for _, result := range started.Started {
		states[result.ID] = deviceState{State: "running"}
		if !result.OK {
			states[result.ID] = deviceState{State: "failed", Error: result.Error}
		}
	}
	s.mu.Lock()
	s.devices = states
	s.mu.Unlock()
	if !config.AutoStart {
		config.AutoStart = true
		_ = s.save(config)
	}
	s.changed(ChangeStatus)
	status := s.Status()
	return &status, nil
}

// Stop stops every simulated device.
func (s *Service) Stop() error {
	s.mu.Lock()
	running := s.process != nil && len(s.devices) > 0
	s.devices = map[string]deviceState{}
	s.mu.Unlock()
	if running {
		if _, err := s.call("agents_stop", nil, startTimeout); err != nil {
			return err
		}
	}
	config := s.Config()
	if config.AutoStart {
		config.AutoStart = false
		_ = s.save(config)
	}
	s.changed(ChangeStatus)
	s.releaseIfIdle()
	return nil
}

// Autostart starts the devices again when they ran as the app closed.
func (s *Service) Autostart() {
	if config := s.Config(); config.AutoStart && len(config.Devices) > 0 {
		if _, err := s.Start(); err != nil {
			s.logf("SNMP devices did not start again: %v", err)
		}
	}
}

func (s *Service) deviceRunning(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.devices[id].State == "running"
}

func deviceOf(config Config, id string) (model.Device, error) {
	index, err := config.findDevice(id)
	if err != nil {
		return model.Device{}, apperr.ValueError(err.Error())
	}
	return config.Devices[index], nil
}

// pointIndex is the position of oid in points.
func pointIndex(points []model.Point, oid string) (int, error) {
	for index, point := range points {
		if point.OID == oid {
			return index, nil
		}
	}
	return -1, apperr.ValueError(fmt.Sprintf("The device has no point %s", oid))
}
