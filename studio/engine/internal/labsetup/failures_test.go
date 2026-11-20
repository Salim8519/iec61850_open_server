package labsetup

import (
	"strings"
	"testing"

	"studio-engine/internal/labnet"
	"studio-engine/internal/netcfg"
)

// Change 0221, trial N1: every lab address the service refuses stops the Run with its own reason;
// before, Run went on as if nothing had happened.
func TestEveryRefusedLabAddressStopsTheRunWithItsReason(t *testing.T) {
	p := newPC()
	p.version = "2"
	s := p.service()
	s.Use(nil)
	call := s.Windows.Call
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		if request.Op != labnet.OpAdd {
			return call(request)
		}
		return labnet.Reply{OK: false, Message: "first", Addresses: []string{}, Version: "2", Failed: map[string]string{
			"192.168.216.1": "Windows did not add 192.168.216.1: The object already exists.",
			"10.20.0.5":     "Windows did not add 10.20.0.5: Access is denied."}}, nil
	}
	err := s.EnsureForRun(card, []string{"a", "b"})
	want := "P545 cannot start on 192.168.216.1: Windows did not add 192.168.216.1: The object already exists. " +
		"D60 cannot start on 10.20.0.5: Windows did not add 10.20.0.5: Access is denied. Stop the other device"
	if err == nil || !strings.HasPrefix(err.Error(), want) {
		t.Fatalf("error %v", err)
	}
}

// One refused address that is not a clash gets the general advice; a reason with its own advice
// (the cable is out) gets none added.
func TestRunAdviceFollowsTheReason(t *testing.T) {
	err := runRefused([]failure{{address: "10.1.1.1", name: "P545", reason: "Windows did not add 10.1.1.1: Access is denied."}})
	if err == nil || err.Error() != "P545 cannot start on 10.1.1.1: Windows did not add 10.1.1.1: Access is denied. "+
		"Press Start again. If it keeps failing, press Repair on Network > This PC lab network." {
		t.Fatalf("error %v", err)
	}
	cable := "“Ethernet” is disconnected (no cable, or its network is switched off), so ScadaBench did not add 10.1.1.1. " +
		"Plug the cable in, then press Start again"
	err = runRefused([]failure{{address: "10.1.1.1", name: "P545", reason: cable}})
	if err == nil || err.Error() != "P545 cannot start on 10.1.1.1: "+cable+"." {
		t.Fatalf("error %v", err)
	}
	if runRefused(nil) != nil {
		t.Fatal("nothing failed")
	}
}

// Every refused real-card address is named, not only the first (before: only the first).
func TestEveryRefusedRealAddressIsNamed(t *testing.T) {
	_, s, _ := onEthernet(t)
	call := s.Windows.Call
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		reply, err := call(request)
		if request.Card != "" && request.Op == labnet.OpAdd {
			reply.Failed = map[string]string{"192.168.40.21": "A device on “Ethernet” already answers at 192.168.40.21; nothing was changed",
				"192.168.40.22": "A device on “Ethernet” already answers at 192.168.40.22; nothing was changed"}
		}
		return reply, err
	}
	err := s.EnsureForRun(eth, []string{"a", "b"})
	if err == nil || !strings.Contains(err.Error(), "P545 cannot start on 192.168.40.21") ||
		!strings.Contains(err.Error(), "D60 cannot start on 192.168.40.22") {
		t.Fatalf("error %v", err)
	}
}

// The service is off (or does not answer) and the addresses are not on the card: Run says so,
// naming them; addresses already on the card need nothing.
func TestRunSaysWhenTheServiceIsOff(t *testing.T) {
	p := newPC()
	s := p.service()
	s.Use(nil)
	s.Windows.State = func() (bool, bool) { return true, false }
	err := s.EnsureForRun(card, []string{"a", "b"})
	if err == nil || !strings.Contains(err.Error(), "The lab network service is off, so ScadaBench could not put "+
		"192.168.216.1 (P545) and 10.20.0.5 (D60) on the ScadaBench Lab card") {
		t.Fatalf("error %v", err)
	}
	p.cardIPs["192.168.216.1"], p.cardIPs["10.20.0.5"] = true, true // still there from before
	if err := s.EnsureForRun(card, []string{"a", "b"}); err != nil {
		t.Fatalf("addresses on the card: %v", err)
	}
	s.Windows.State = func() (bool, bool) { return true, true }
	s.Windows.Call = func(labnet.Request) (labnet.Reply, error) { return labnet.Reply{}, labnet.ErrNoService }
	delete(p.cardIPs, "10.20.0.5")
	if err := s.EnsureForRun(card, []string{"a", "b"}); err == nil || !strings.Contains(err.Error(), "does not answer") ||
		!strings.Contains(err.Error(), "10.20.0.5 (D60)") || strings.Contains(err.Error(), "192.168.216.1") {
		t.Fatalf("not answering: %v", err)
	}
}

// Stop with the service off: the addresses that stay are named with the reason, journaled as
// failed and shown on the lab page; before, Release did nothing and said nothing.
func TestStopNamesTheAddressesThatStay(t *testing.T) {
	p := newPC()
	s := p.service()
	s.Journal = netcfg.NewJournal(t.TempDir())
	s.Use(nil)
	p.cardIPs["192.168.216.1"] = true
	p.reserved = map[string]string{"10.9.9.1": "Station gateway"}
	p.cardIPs["10.9.9.1"] = true
	s.Windows.State = func() (bool, bool) { return true, false }
	err := s.Release()
	if err == nil || !strings.Contains(err.Error(), "192.168.216.1 on ScadaBench Lab (the lab network service is off)") ||
		strings.Contains(err.Error(), "10.9.9.1") {
		t.Fatalf("error %v", err)
	}
	records, _ := s.Journal.Records()
	if event, _ := records[len(records)-1].Get("event"); event != "failed" {
		t.Fatalf("journal %v", records)
	}
	status, _ := s.Status(nil)
	if !strings.Contains(status.Problem, "cannot take 192.168.216.1 off the card") || strings.Contains(status.Problem, "10.9.9.1") {
		t.Fatalf("problem %q", status.Problem)
	}
	s.Windows.State = func() (bool, bool) { return true, true }
	if err := s.Release(); err != nil || p.cardIPs["192.168.216.1"] {
		t.Fatalf("service back: %v %v", err, p.cardIPs)
	}
}

// A removal the service refuses is named with its reason.
func TestStopNamesARefusedRemoval(t *testing.T) {
	p := newPC()
	p.version = "2"
	s := p.service()
	s.Use(nil)
	p.cardIPs["192.168.216.1"], p.cardIPs["10.20.0.5"] = true, true
	call := s.Windows.Call
	s.Windows.Call = func(request labnet.Request) (labnet.Reply, error) {
		if request.Op != labnet.OpRemove {
			return call(request)
		}
		return labnet.Reply{OK: false, Message: "x", Addresses: []string{"10.20.0.5"}, Version: "2",
			Failed: map[string]string{"10.20.0.5": "Windows did not remove 10.20.0.5: Access is denied."}}, nil
	}
	err := s.Release()
	if err == nil || !strings.Contains(err.Error(), "10.20.0.5 on ScadaBench Lab (Windows did not remove 10.20.0.5: Access is denied)") ||
		strings.Contains(err.Error(), "192.168.216.1") {
		t.Fatalf("error %v", err)
	}
}
