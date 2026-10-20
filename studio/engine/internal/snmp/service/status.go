package service

import (
	"strconv"

	"studio-engine/internal/snmp/profiles"
)

// Status is GET /tools/snmp: what runs, for the page and the Tools card.
type Status struct {
	Role         string         `json:"role"`
	Engine       string         `json:"engine"` // off, running or failed
	Message      string         `json:"message"`
	Devices      []DeviceStatus `json:"devices"`
	Running      int            `json:"running"`
	Reader       ReaderStatus   `json:"reader"`
	Traps        TrapStatus     `json:"traps"`
	RequestsLast int64          `json:"requests_last"`
	TrapsLast    int64          `json:"traps_last"`
}

// DeviceStatus is one simulated device in the list.
type DeviceStatus struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Profile string `json:"profile"`
	Kind    string `json:"kind"`
	Address string `json:"address"`
	State   string `json:"state"` // running, stopped or failed
	Error   string `json:"error"`
	Points  int    `json:"points"`
}

// ReaderStatus is the Read devices state.
type ReaderStatus struct {
	State     string `json:"state"` // disconnected, connecting, walking, connected or failed
	Message   string `json:"message"`
	Target    string `json:"target"`
	System    []any  `json:"system"`
	EngineID  string `json:"engine_id"`
	WalkCount int64  `json:"walk_count"`
	Values    int    `json:"values"`
	Polled    int    `json:"polled"`
	EveryMS   int    `json:"every_ms"`
	Problem   string `json:"problem"`
}

// TrapStatus is the trap receiver.
type TrapStatus struct {
	Listening bool   `json:"listening"`
	Address   string `json:"address"`
	Message   string `json:"message"`
}

// Status reads the current state.
func (s *Service) Status() Status {
	config := s.Config()
	s.mu.Lock()
	defer s.mu.Unlock()
	status := Status{Role: config.Role, Engine: "off", Devices: []DeviceStatus{},
		RequestsLast: s.Requests.Sequence(), TrapsLast: s.Traps.Sequence()}
	if s.process != nil {
		status.Engine = "running"
	}
	if s.failure != "" {
		status.Engine, status.Message = "failed", s.failure
	}
	for _, device := range config.Devices {
		state := s.devices[device.ID]
		item := DeviceStatus{ID: device.ID, Name: device.Name, Profile: device.Profile, Kind: profiles.Names[device.Profile],
			Address: device.IP + ":" + strconv.Itoa(device.Port), State: "stopped", Error: state.Error,
			Points: len(device.Points)}
		if state.State != "" {
			item.State = state.State
		}
		if item.State == "running" {
			status.Running++
		}
		status.Devices = append(status.Devices, item)
	}
	r := s.reader
	status.Reader = ReaderStatus{State: r.state, Message: r.message, Target: r.describe(), System: r.system,
		EngineID: r.engineID, WalkCount: r.walkCount, Values: len(r.order), Polled: len(r.polled),
		EveryMS: config.Reader.EveryMS, Problem: r.problem}
	if status.Reader.System == nil {
		status.Reader.System = []any{}
	}
	status.Traps = TrapStatus{Listening: s.listening, Message: s.trapText,
		Address: config.Reader.TrapIP + ":" + strconv.Itoa(config.Reader.TrapPort)}
	return status
}
