package service

import "studio-engine/internal/apperr"

// StartDevice starts one device; the devices already running keep running untouched
// (change 0234, owner: "a button to run one snmp device instead of all").
func (s *Service) StartDevice(id string) (*Status, error) {
	config := s.Config()
	device, err := deviceOf(config, id)
	if err != nil {
		return nil, err
	}
	if err := CheckDevice(device, config.Devices); err != nil {
		return nil, apperr.ValueError(err.Error())
	}
	lab := s.onLab()
	placed, err := place(device, lab)
	if err == nil && lab {
		s.markStarting(device.ID, placed.IP) // a Release meanwhile keeps it (ADR-0141, trial N5)
		err = s.ensureOnCard(placed)
	}
	defer s.clearStarting()
	if err != nil {
		return nil, err
	}
	answer, err := s.call("agent_start", map[string]any{"device": placed}, startTimeout)
	if err != nil {
		return nil, err
	}
	var started struct {
		Started struct {
			OK    bool   `json:"ok"`
			Error string `json:"error"`
		} `json:"started"`
	}
	if err := FromPy(answer, &started); err != nil {
		return nil, err
	}
	state := deviceState{State: "running"}
	if !started.Started.OK {
		state = deviceState{State: "failed", Error: started.Started.Error}
	}
	s.mu.Lock()
	if s.devices == nil {
		s.devices = map[string]deviceState{}
	}
	if s.bound == nil {
		s.bound = map[string]string{}
	}
	s.devices[id], s.bound[id] = state, placed.IP
	s.mu.Unlock()
	s.changed(ChangeStatus)
	status := s.Status()
	return &status, nil
}

// StopDevice stops one device; the others keep running. When none is left it is Stop.
func (s *Service) StopDevice(id string) (*Status, error) {
	if _, err := deviceOf(s.Config(), id); err != nil {
		return nil, err
	}
	s.mu.Lock()
	_, known := s.devices[id]
	others := 0
	for other, state := range s.devices {
		if other != id && state.State == "running" {
			others++
		}
	}
	s.mu.Unlock()
	if others == 0 {
		if err := s.Stop(); err != nil {
			return nil, err
		}
		status := s.Status()
		return &status, nil
	}
	if known {
		if _, err := s.call("agent_stop", map[string]any{"device": id}, startTimeout); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	delete(s.devices, id)
	delete(s.bound, id)
	s.mu.Unlock()
	s.changed(ChangeStatus)
	s.releaseCard() // only addresses no running device uses any more (ADR-0127)
	status := s.Status()
	return &status, nil
}
