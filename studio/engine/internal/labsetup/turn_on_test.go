package labsetup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"studio-engine/internal/netcfg"
)

// A stopped network service is turned back on before Start puts the approved addresses on the
// real card: one Windows prompt running "start" (ADR-0136). A cancelled prompt stops the Run.
func TestStartTurnsTheStoppedServiceBackOn(t *testing.T) {
	_, s, requests := onEthernet(t)
	running, asked := false, []string{}
	s.Windows.State = func() (bool, bool) { return true, running }
	s.Program = filepath.Join(t.TempDir(), "scadabench_labnet.exe")
	os.WriteFile(s.Program, nil, 0o666)
	s.Directory, s.NewID = t.TempDir(), func() string { return "id" }
	s.Elevator = netcfg.ElevatorFunc(func(program string, arguments []string) (uint32, error) {
		asked = append(asked, arguments[0])
		running = true
		return 0, os.WriteFile(arguments[2], []byte(`{"ok":true,"message":"running again"}`), 0o666)
	})
	if err := s.EnsureForRun(eth, []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(asked, ",") != "start" || len(*requests) != 1 {
		t.Fatalf("asked %v, requests %v", asked, *requests)
	}
	if err := s.EnsureForRun(eth, []string{"a"}); err != nil || len(asked) != 1 {
		t.Fatalf("running service asked again: %v %v", err, asked)
	}

	running = false
	s.Elevator = netcfg.ElevatorFunc(func(string, []string) (uint32, error) { return 0, netcfg.ErrElevationCancelled })
	if err := s.EnsureForRun(eth, []string{"a"}); err == nil || !strings.Contains(err.Error(), "permission") {
		t.Fatalf("cancelled prompt: %v", err)
	}
}

// A project on This PC only, or a real card without approved addresses, never asks Windows.
func TestNoPromptWhenTheRunDoesNotNeedTheService(t *testing.T) {
	_, s, _ := onEthernet(t)
	s.Windows.State = func() (bool, bool) { return true, false }
	s.Elevator = netcfg.ElevatorFunc(func(string, []string) (uint32, error) {
		t.Fatal("Windows was asked")
		return 0, nil
	})
	if err := s.EnsureForRun("", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureForRun("{WIFI}", []string{"a"}); err != nil {
		t.Fatal(err)
	}
}
