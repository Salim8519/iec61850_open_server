package labsetup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"studio-engine/internal/apperr"
	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

// helperOutcome is the result file scadabench_labnet.exe writes.
type helperOutcome struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	Installed *struct {
		Card    string `json:"card"`
		Created bool   `json:"created"`
	} `json:"installed"`
}

// Install is POST /network/lab/install: the one administrator prompt that creates the card and
// the service (or repairs them: an existing card is kept).
func (s *Service) Install() (Status, error) {
	s.change.Lock()
	defer s.change.Unlock()
	outcome, err := s.elevate("install")
	if err != nil {
		return Status{}, err
	}
	if outcome.Installed != nil && outcome.Installed.Card != "" {
		event := "skipped" // repaired: the card already existed and stays unowned by this run
		if outcome.Installed.Created {
			event = "applied"
		}
		s.journal(event, "lab_card", "", "", outcome.Installed.Card, outcome.Message)
	}
	if !outcome.OK {
		return Status{}, apperr.New(outcome.Message)
	}
	s.call(labnet.Request{Op: labnet.OpStatus}) // the repaired service's version (ADR-0128)
	return s.Status(nil)
}

// Remove is POST /network/lab/remove: the project leaves the card first (Run stopped), then one
// administrator prompt removes the service, its program and the card.
func (s *Service) Remove(leave func() error) (Status, error) {
	s.change.Lock()
	defer s.change.Unlock()
	if len(s.Runtime.Running()) > 0 {
		return Status{}, plain("Stop the simulation before removing the lab network card.")
	}
	card := s.Windows.Card()
	if mode, guid := s.Projects.Network(); mode == "lab" && card != "" && strings.EqualFold(guid, card) {
		if err := leave(); err != nil {
			return Status{}, err
		}
	}
	outcome, err := s.elevate("uninstall")
	if err != nil {
		return Status{}, err
	}
	event := "removed"
	if !outcome.OK {
		event = "failed"
	}
	s.journal(event, "lab_card", "", "", card, outcome.Message)
	if !outcome.OK {
		return Status{}, apperr.New(outcome.Message)
	}
	return s.Status(nil)
}

// elevate runs scadabench_labnet.exe <mode> --result <file> through the Windows prompt.
func (s *Service) elevate(mode string) (helperOutcome, error) {
	var outcome helperOutcome
	if _, err := os.Stat(s.Program); err != nil {
		return outcome, apperr.New("The lab network program (scadabench_labnet.exe) is missing from this installation.")
	}
	folder := filepath.Join(s.Directory, "network", "requests")
	if err := os.MkdirAll(folder, 0o777); err != nil {
		return outcome, err
	}
	result := filepath.Join(folder, "lab-"+mode+"-"+s.NewID()+".result.json")
	if _, err := s.Elevator.Elevate(s.Program, []string{mode, "--result", result}); err != nil {
		if errors.Is(err, netcfg.ErrElevationCancelled) {
			return outcome, apperr.New("Windows did not get permission, so nothing was changed.")
		}
		return outcome, err
	}
	data, err := os.ReadFile(result)
	if err != nil {
		return outcome, apperr.New("The lab network program finished without a result; nothing was recorded as changed.")
	}
	os.Remove(result)
	if err := json.Unmarshal(data, &outcome); err != nil {
		return outcome, apperr.New("The lab network program's result could not be read.")
	}
	return outcome, nil
}
