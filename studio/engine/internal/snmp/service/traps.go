package service

import (
	"fmt"
	"net/netip"
	"time"

	"studio-engine/internal/apperr"
	"studio-engine/internal/snmp/model"
	"studio-engine/internal/snmp/proto"
)

// TrapSettings are what the page sends to start the trap receiver.
type TrapSettings struct {
	IP    string       `json:"ip"`
	Port  int          `json:"port"`
	Users []model.User `json:"users"`
}

// Listen starts the trap receiver; traps and informs appear in the trap log.
func (s *Service) Listen(settings TrapSettings) (Status, error) {
	address, err := netip.ParseAddr(settings.IP)
	if err != nil || !address.Is4() {
		return Status{}, apperr.ValueError("Choose where to listen: this PC only (127.0.0.1) or all network cards (0.0.0.0)")
	}
	if settings.Port < 1 || settings.Port > 65535 {
		return Status{}, apperr.ValueError("The trap port must be 1 to 65535 (usually 162)")
	}
	if settings.Users == nil {
		settings.Users = []model.User{}
	}
	for _, user := range settings.Users {
		if problem := proto.CheckUser(user); problem != "" {
			return Status{}, apperr.ValueError(problem)
		}
	}
	config := s.Config()
	config.Reader.TrapIP, config.Reader.TrapPort, config.Reader.TrapUsers = settings.IP, settings.Port, settings.Users
	_, err = s.call("traps_listen", map[string]any{"ip": settings.IP, "port": settings.Port, "users": settings.Users,
		"engine_id": config.Reader.EngineID}, 30*time.Second)
	s.mu.Lock()
	if err != nil {
		s.listening, s.trapText = false, err.Error()
	} else {
		s.listening, s.trapText = true, fmt.Sprintf("Listening on %s port %d", settings.IP, settings.Port)
	}
	s.mu.Unlock()
	config.Reader.TrapsOn = err == nil
	if saveErr := s.save(config); saveErr != nil && err == nil {
		err = saveErr
	}
	s.changed(ChangeStatus)
	if err != nil {
		s.releaseIfIdle()
		return s.Status(), apperr.ValueError(err.Error())
	}
	return s.Status(), nil
}

// StopListening closes the trap port.
func (s *Service) StopListening() Status {
	s.mu.Lock()
	listening := s.listening && s.process != nil
	s.listening, s.trapText = false, ""
	s.mu.Unlock()
	if listening {
		s.call("traps_stop", nil, 10*time.Second)
	}
	config := s.Config()
	if config.Reader.TrapsOn {
		config.Reader.TrapsOn = false
		_ = s.save(config)
	}
	s.changed(ChangeStatus)
	s.releaseIfIdle()
	return s.Status()
}

// AutoListen opens the trap port again when it was open as the app closed.
func (s *Service) AutoListen() {
	config := s.Config()
	if !config.Reader.TrapsOn {
		return
	}
	if _, err := s.Listen(TrapSettings{IP: config.Reader.TrapIP, Port: config.Reader.TrapPort,
		Users: config.Reader.TrapUsers}); err != nil {
		s.logf("The SNMP trap receiver did not start again: %v", err)
	}
}

// SetRole switches the tool between Simulate devices and Read devices; nothing stops.
func (s *Service) SetRole(role string) (Status, error) {
	if role != RoleSimulate && role != RoleRead {
		return Status{}, apperr.ValueError("Choose Simulate devices or Read devices")
	}
	config := s.Config()
	config.Role = role
	if err := s.save(config); err != nil {
		return Status{}, err
	}
	s.changed(ChangeStatus)
	return s.Status(), nil
}
