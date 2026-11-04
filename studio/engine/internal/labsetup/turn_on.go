package labsetup

import (
	"strings"

	"studio-engine/internal/apperr"
)

// The network service can be installed but stopped: another program or a Windows setting may
// switch it to manual start. ScadaBench then turns it back on itself (ADR-0136): the address
// plan's Windows prompt sets it up (netcfg service_repair.go), and Start asks Windows once before
// it continues.

// turnOnForRun turns the stopped service back on before a Run that needs it: one Windows prompt
// runs scadabench_labnet.exe start. Without a prompt (tests) nothing changes.
func (s *Service) turnOnForRun() error {
	if installed, running := s.Windows.State(); !installed || running || s.Elevator == nil {
		return nil
	}
	outcome, err := s.elevate("start")
	if err != nil {
		return err
	}
	event := "applied"
	if !outcome.OK {
		event = "failed"
	}
	s.journal(event, "lab_service", "", "", "", outcome.Message)
	if !outcome.OK {
		return apperr.New(outcome.Message)
	}
	return nil
}

// needsService: a Run on the lab card, or on a real card with approved addresses, uses the service.
func (s *Service) needsService(guid string) bool {
	if guid == "" {
		return false
	}
	if card := s.Windows.Card(); card != "" && strings.EqualFold(card, guid) {
		return true
	}
	return s.Windows.Approved != nil && len(s.Windows.Approved()[strings.ToUpper(guid)]) > 0
}
